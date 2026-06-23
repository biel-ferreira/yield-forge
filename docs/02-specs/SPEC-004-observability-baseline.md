# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Observability Baseline (OpenTelemetry)                 |
| Feature ID   | SPEC-004 (foundational)                                |
| Related PRD  | [PRD.md](../01-product/PRD.md) — §10 NFR Observability, G11, p95 latency metric |
| Related ADRs | [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md) (free/self-hosted OTel backend), [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) |
| Version      | 0.1.0                                                  |
| Status       | Approved                                               |
| Plan         | PLAN-004 (authored next via /plan-new 004)             |

---

## 2. Overview

### Purpose

Give the application **production-grade observability**: distributed **traces**,
**metrics**, and **log/trace correlation** via OpenTelemetry (OTel), wired as
cross-cutting infrastructure so every later feature is observable by default. This
spec stands up the OTel SDK, instruments the HTTP server and the database, correlates
the existing `slog` logs with trace IDs, and exports through a **vendor-neutral OTLP**
pipeline that is **safe to run with no backend at all** (zero cost).

It also defines the **instrumentation seam** — an injected `Tracer`/`Meter` plus
conventions — so SPEC-005 (LLM) can record each AI interaction end-to-end
(prompt/model/latency/token-cost) and SPEC-006 (market data) can record ingestion
metrics, with no rework (PRD §10).

### Business Value

- **G11 — production-grade engineering.** Traces + metrics are the difference between
  "it's slow somewhere" and "the DB query in `GetHoldings` is the p95 spike."
- **Verifies the SLOs.** The PRD targets **< 300 ms p95** for non-AI reads; you cannot
  hold a latency target you do not measure. This baseline provides that measurement.
- **Enables AI debuggability + Context Engineering.** PRD §10 requires every AI
  interaction be traceable (prompt, model, latency, cost, outcome). This spec lays the
  rails; SPEC-005 rides them.
- **Zero cost (ADR-0003).** OTel SDK is free; the OTLP exporter targets a free-tier or
  self-hosted backend (Grafana Cloud free tier / Jaeger / Prometheus / a local
  collector) — selected purely by config, and **disabled by default** so nothing is
  required to run locally or in CI.

### Scope

**In scope:** OTel SDK bootstrap (Tracer/Meter providers + a `Resource` describing the
service); a configurable OTLP exporter that no-ops without an endpoint; HTTP server
instrumentation (request traces + duration/count metrics); W3C trace-context
propagation; `slog`↔trace correlation (trace_id/span_id on log lines); database
query instrumentation; graceful provider shutdown (flush on exit); the
`Tracer`/`Meter` injection seam + instrumentation conventions; config; tests;
CHANGELOG/README/lesson.

**Out of scope (owned by later specs / ops):**
- AI-call spans + token/cost metrics → **SPEC-005** (emitted via this spec's Meter;
  money cost follows the `int64` convention).
- Market-data ingestion success-rate & freshness metrics → **SPEC-006**.
- Feature-specific business spans/metrics → each feature spec (this spec instruments
  only the cross-cutting HTTP + DB layers and provides the seam).
- Choosing/operating a concrete hosted backend, dashboards, and alerting → deploy-time
  / ops (ADR-0003 open item).
- Replacing the logging pipeline with the OTel **logs** SDK → we keep `slog` and add
  trace correlation (§14-D4); a future migration is possible but not now.
- Profiling (pprof/continuous profiling) → future.

---

## 3. Functional Requirements

> Foundational, like SPEC-001/002. SPEC-004 implements no PRD *feature* FR directly;
> it satisfies the §10 Observability NFR and enables the AI-traceability the feature
> specs depend on.

### FR-401 — OTel SDK Bootstrap & Resource

The application initialises OpenTelemetry at startup with a `Resource` identifying the
service, and providers that are injected (never global-only) and cleanly shut down.

**Acceptance Criteria**
- [ ] A `platform/observability` (or `telemetry`) package exposes a single
      `Setup(ctx, cfg) (shutdown func(ctx) error, error)` that builds the Tracer and
      Meter providers and returns a shutdown that flushes + closes them.
- [ ] The OTel `Resource` carries at least `service.name`, `service.version` (from
      buildinfo), and `deployment.environment` (`APP_ENV`).
- [ ] `Setup` is called once in `cmd/api/main.go`; the shutdown runs on graceful exit
      **before** the process returns (telemetry is flushed, not dropped).
- [ ] The global OTel propagator and (optionally) providers are set so library
      instrumentation works, but the app also holds references for explicit use.

### FR-402 — Configurable, Zero-Cost-Safe Exporter

Telemetry export is environment-driven and **never required** for the app to run.

**Acceptance Criteria**
- [ ] When `OTEL_EXPORTER_OTLP_ENDPOINT` is unset, exporting is a **no-op**: the app
      starts and serves normally, with telemetry collected but not shipped (or dropped
      cheaply). No crash, no error, no blocking (BR-401).
- [ ] When the endpoint is set, traces (and metrics) are exported via **OTLP** to it;
      an optional `OTEL_EXPORTER_OTLP_HEADERS` carries backend auth (a secret from the
      env). A dev/debug `stdout` exporter is available behind config.
- [ ] An exporter or backend being unreachable **degrades gracefully** — it must not
      take down request handling (export happens in the background; failures are logged
      at most, never propagated to the request path).
- [ ] Sampling is configurable (`OTEL_TRACES_SAMPLER` / ratio) with a sensible default
      (parent-based; sample-all in dev, ratio in prod).

### FR-403 — HTTP Server Instrumentation (traces + metrics)

Every HTTP request produces a span and contributes to request metrics, reusing the
existing middleware chain.

**Acceptance Criteria**
- [ ] Incoming requests are wrapped with `otelhttp` (or equivalent) so each produces a
      server span named by the **route pattern** (not the raw path, to avoid cardinality
      blowups), with method, status, and duration.
- [ ] HTTP server metrics are recorded: request **duration histogram** and request
      **count** by route + status, enabling the p95-latency SLO check.
- [ ] Liveness/readiness probes (`/healthz`, `/readyz`) are low-noise (not sampled into
      every trace, or clearly marked) so they don't drown the signal.
- [ ] The span integrates with the existing `requestID`/`logRequests` middleware — the
      request id and trace id appear together (FR-405).

### FR-404 — Trace-Context Propagation

Trace context flows in and out using the W3C standard.

**Acceptance Criteria**
- [ ] The global propagator is **W3C Trace Context** (+ Baggage); an incoming
      `traceparent` header continues the trace, and outgoing calls (DB, later LLM /
      market-data) carry context via `context.Context` (BR-405).
- [ ] A request with no incoming trace starts a new root trace.

### FR-405 — Log ↔ Trace Correlation

Structured logs carry the active trace identifiers so a log line can be pivoted to its
trace.

**Acceptance Criteria**
- [ ] When a span is active, `slog` records include `trace_id` and `span_id` (e.g. via
      a handler/middleware that reads them from `context`).
- [ ] The per-request log line (`logRequests`) already carries `request_id`; it now also
      carries `trace_id` when present, tying the two correlation IDs together.
- [ ] No telemetry plumbing leaks into business code — correlation is done at the
      logging/transport edge (BR-403).

### FR-406 — Database Instrumentation

Database work is visible in traces (PRD §10: traces across API → application → DB).

**Acceptance Criteria**
- [ ] DB queries executed through the pool produce child spans (operation + a
      **non-sensitive** statement label) under the request span, so a slow query is
      attributable to its request.
- [ ] No SQL **arguments / parameter values** are recorded (no PII/secrets — BR-402);
      statement text is recorded only if it is parameterised (no inlined values).
- [ ] The instrumentation is applied where the pool is built (`platform/database`),
      keeping feature/repository code free of OTel imports.

### FR-407 — Instrumentation Seam for Features

Later features can add spans/metrics through an injected, swappable seam — not by
reaching for globals.

**Acceptance Criteria**
- [ ] A `Tracer` and a `Meter` (scoped to the app) are obtainable and injectable into
      services/adapters that need custom instrumentation, so SPEC-005 can record AI
      spans (prompt/model/latency/**token-cost**) and SPEC-006 ingestion metrics.
- [ ] The conventions are documented: span naming, the no-PII/no-secrets rule, where
      spans live (adapter/transport boundaries, not the pure domain), and how money-
      valued metrics (e.g. AI cost) follow the `int64` convention (CLAUDE.md).
- [ ] A feature core (pure domain) still imports **no** OTel types; instrumentation is
      added at the edges or via the injected seam (BR-403).

### FR-408 — Config, Docs & Lesson

**Acceptance Criteria**
- [ ] New env vars documented in `.env.example` (endpoint, headers, sampler/ratio,
      service name override, enable/disable, dev stdout toggle).
- [ ] `CHANGELOG.md` `[Unreleased]` records the observability baseline.
- [ ] `README.md` documents how to run with a local collector / Jaeger and how it
      no-ops without one.
- [ ] On close: SPEC-004 + PLAN-004 flipped to Done, indexes updated, and a PT-BR
      lesson `docs/lessons/SPEC-004-aula.html` produced.

---

## 4. User Flows

> The "user" of SPEC-004 is the **developer/operator**.

### Flow 1 — Run with no backend (zero cost)
1. Developer runs the app with `OTEL_EXPORTER_OTLP_ENDPOINT` unset.
2. App starts normally; spans/metrics are collected but not shipped (no-op exporter).
3. Requests serve as before — observability adds no hard dependency.

### Flow 2 — Run with a local collector / Jaeger
1. Developer starts a local OTLP collector (or Jaeger all-in-one) and sets the endpoint.
2. App exports traces + metrics; a request to `/auth/me` shows a trace `HTTP GET
   /auth/me → DB query` with timings, and a `trace_id` appears in that request's log line.

### Flow 3 — Backend unreachable (degrade gracefully)
1. The configured endpoint is down.
2. Export fails in the background and is logged at most; **request handling is
   unaffected** (no added latency or errors on the request path).

### Flow 4 — Graceful shutdown flushes telemetry
1. `SIGINT`/`SIGTERM` triggers shutdown.
2. The OTel providers flush pending spans/metrics within a bounded timeout, then close,
   before the process exits — recent telemetry is not lost.

---

## 5. Business Rules (Architectural)

- **BR-401 — Telemetry is never required to run.** No endpoint configured → a no-op
  pipeline; the app behaves exactly as without observability. Export failures never
  reach the request path (zero-cost + reliability).
- **BR-402 — No secrets or PII in telemetry.** Spans, attributes, metric labels, and
  events must never contain passwords, session tokens, raw emails, full DSNs/credentials,
  or SQL argument values. Reuse the redaction posture from SPEC-002/003.
- **BR-403 — Instrumentation is cross-cutting and edge-placed.** OTel lives in
  `platform/observability`, `platform/database`, and `transport/http`. A feature's pure
  **domain/service core imports no OTel types**; custom spans use the injected seam
  (FR-407), honouring the dependency direction (SPEC-001 BR-001/002).
- **BR-404 — Vendor-neutral.** Only the OTel API + OTLP are used; no vendor SDK lock-in,
  so the backend is a config swap (ADR-0003).
- **BR-405 — Context carries trace state.** Trace context propagates via
  `context.Context` (our ctx-first convention), never via globals or thread-locals.
- **BR-406 — Bounded, configurable cost.** Sampling is configurable; cardinality is
  controlled (route patterns, not raw paths; bounded attribute sets) so telemetry volume
  stays within free-tier limits.

---

## 6. Domain Model

No domain entities (an infrastructure spec, like SPEC-001/002). SPEC-004 introduces only
the **observability conventions** future code follows:

| Convention            | Rule                                                            |
| --------------------- | -------------------------------------------------------------- |
| Span names            | Low-cardinality (route pattern / operation), not raw paths/ids |
| Attributes            | Bounded set; never secrets/PII (BR-402)                        |
| Correlation IDs       | `trace_id` + `span_id` on logs; alongside the existing `request_id` |
| Money-valued metrics  | `int64` minor units / basis points (CLAUDE.md), e.g. future AI cost |
| Placement             | Edges (transport/adapters) or injected seam — never the pure domain |

The only Go types added are the infrastructure provider/seam in
`internal/platform/observability` — not domain types.

---

## 7. API Specification

No new endpoints, and the existing endpoints (`/healthz`, `/readyz`, `/version`,
`/auth/*`) are **unchanged** in contract — they merely become traced/measured.

Telemetry leaves via **OTLP push** to the configured collector, so **no `/metrics`
scrape endpoint is exposed** by default (avoids a public surface). If a Prometheus
*pull* model is ever chosen, exposing an authenticated `/metrics` would be a separate,
explicit decision.

---

## 8. Data Storage

None. SPEC-004 introduces no tables, migrations, or schema. Telemetry is exported to an
external backend (out of process), not persisted by the app.

---

## 9. Edge Cases

| Scenario | Expected behaviour |
| -------- | ------------------ |
| `OTEL_EXPORTER_OTLP_ENDPOINT` unset | No-op exporter; app runs normally (BR-401). |
| Backend unreachable / slow | Export retries/drops in the background; request path unaffected. |
| Malformed `OTEL_*` config | Fail fast at startup with a clear error (config is validated like the rest). |
| High request volume | Sampling + low-cardinality names keep volume bounded (BR-406). |
| Shutdown with pending spans | Flushed within a bounded timeout, then providers close; then exit. |
| Health/probe spam | Probes are low-noise (de-prioritised/!sampled) so they don't dominate traces. |
| A span started but never ended (bug) | Caught by review/tests; spans use `defer span.End()` at the edge. |

---

## 10. Security Considerations

- **No secrets/PII in telemetry** (BR-402) — the single most important rule here;
  enforced by review + the no-args DB instrumentation and redaction conventions.
- **Backend credentials** (`OTEL_EXPORTER_OTLP_HEADERS`, e.g. an API key) are secrets
  read only from the env, never committed; placeholdered in `.env.example`.
- **Transport** to the backend uses TLS in hosted environments (OTLP/gRPC or HTTP over
  TLS); plain only for a local collector in dev.
- **No new public surface** — OTLP is push; no scrape endpoint exposed (see §7).
- Trace ids are not secrets, but they are correlation handles — fine to log.

---

## 11. Observability

This spec **is** the observability baseline. It establishes:
- **Traces:** server spans (HTTP) → DB spans, W3C-propagated, OTLP-exported.
- **Metrics:** HTTP request duration histogram + count (enabling the p95 SLO);
  the `Meter` seam for feature metrics (AI cost/latency in SPEC-005, ingestion in 006).
- **Logs:** existing `slog` baseline (SPEC-001) now correlated with `trace_id`/`span_id`.
- **The end-to-end AI-traceability rails** (FR-407) that PRD §10 requires, populated by
  SPEC-005.

What it does **not** yet emit: AI token/cost metrics, ingestion freshness — those are
their owning specs, using this spec's Meter.

---

## 12. Testing Strategy

### Unit Tests
- `Setup` returns working providers + a shutdown; with no endpoint it builds a no-op
  pipeline and never errors (BR-401).
- HTTP instrumentation: a request through the wrapped handler produces a span with the
  expected name/attributes — asserted with an **in-memory span exporter / recorder**
  (no network).
- Log correlation: with an active span, a `slog` record carries `trace_id`/`span_id`.
- Config: new `OTEL_*` vars parse; defaults applied; bad values fail fast.
- Propagation: an incoming `traceparent` is continued; absent → new root.

### Integration Tests (gated like SPEC-002/003)
- With a real DB (`TEST_DATABASE_URL`), a request that hits the database produces a
  parent HTTP span with a child DB span.
- Graceful shutdown flushes a pending span to an in-memory/stub exporter.
- (Optional, behind config) export to a local OTLP collector if one is available.

### Quality gate
- `go build`/`go vet`/`gofmt` clean; unit tests pass with no backend; dependency rule
  holds (feature cores import no OTel); no secrets/PII asserted in emitted attributes.

---

## 13. Definition of Done

- [ ] `internal/platform/observability` with `Setup` (Tracer + Meter providers,
      `Resource`, no-op-safe OTLP exporter) + graceful shutdown wired in `cmd/api`.
- [ ] HTTP server instrumented (route-named spans + duration/count metrics); probes
      low-noise; integrated with existing request-id/logging middleware.
- [ ] W3C trace-context propagation; `slog` records correlated with `trace_id`/`span_id`.
- [ ] Database queries produce child spans (no arg values/PII).
- [ ] `Tracer`/`Meter` injection seam + documented instrumentation conventions (FR-407).
- [ ] Config env-driven + `.env.example`; runs no-op without a backend; degrades
      gracefully when the backend is down.
- [ ] Unit tests (in-memory exporter) + gated integration test pass; build/vet/fmt clean.
- [ ] `CHANGELOG.md` + `README.md` updated; SPEC-004 + PLAN-004 flipped to Done; indexes
      updated; PT-BR lesson produced.
- [ ] PLAN-004 followed; PR reviewed (hexagonal + go-correctness) and merged.

---

## 14. Decisions (resolved)

> Confirmed with the project owner before PLAN-004. These are now binding.

- **D1 — Traces + metrics + log-correlation.** ✅ All three signals now (PRD §10), so the
  p95-latency SLO is measurable from the baseline; no second pass.
- **D2 — OTLP exporter, config-gated, disabled-by-default; stdout for dev.** ✅
  Vendor-neutral (ADR-0003), one pipeline for traces + metrics, **zero required backend**
  (no-op without an endpoint). Prometheus-pull/`/metrics` rejected (extra public surface).
- **D3 — `otelhttp` contrib middleware** for HTTP instrumentation. ✅ The standard way to
  instrument `net/http`; hand-rolled spans rejected. Adds the OTel contrib dep (justified
  by G11/ADR-0003).
- **D4 — Keep `slog`, inject `trace_id`/`span_id`.** ✅ Preserves the SPEC-001 logging
  baseline; the OTel logs SDK/bridge is deferred (newer/heavier, little MVP gain).
- **D5 — Instrument the `database/sql` handle at the pool (`otelsql`).** ✅ DB spans for
  free with no OTel in repositories; per-repository manual spans rejected.

---

## 15. Open Questions (deferred, not blocking)

- Concrete hosted backend (Grafana Cloud free tier vs self-hosted Jaeger/Prometheus +
  Tempo) — decided at deploy time (ADR-0003 open item); SPEC-004 only requires "an OTLP
  endpoint."
- Whether to add a thin **ADR** recording the OTel/OTLP/`otelhttp` choice (ADR-0003
  already blesses OpenTelemetry; a dedicated ADR may be overkill) — decide during PLAN.
- Default sampling ratio for prod once real traffic exists — start conservative, tune
  with data.
- Continuous profiling (pprof / Pyroscope) — future, not part of the baseline.
- Migrating logs onto the OTel logs SDK once it matures — future (D4 revisit).
