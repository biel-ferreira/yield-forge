# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Observability Baseline (OpenTelemetry)                       |
| Related Feature | Foundational — traces + metrics + log correlation            |
| Related Spec    | [SPEC-004](../02-specs/SPEC-004-observability-baseline.md)    |
| Version         | 0.1.0                                                        |
| Status          | Done                                                         |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-19                                                   |

---

## 2. Objective

### Goal

Wire OpenTelemetry into the app as cross-cutting infrastructure: a no-op-safe OTLP
pipeline (traces + metrics), HTTP and database instrumentation, and `slog`↔trace
correlation — plus the injected `Tracer`/`Meter` seam later specs use. All behind
config, **disabled by default** so the app still runs at zero cost with no backend.

### Expected Outcome

With no `OTEL_EXPORTER_OTLP_ENDPOINT`, the app runs exactly as today (no-op exporter).
With an endpoint (local collector / Jaeger / Grafana free tier), a request to
`/auth/me` produces a trace `HTTP GET /auth/me → DB query` with timings, HTTP request
metrics (duration histogram + count) are recorded, and that request's log line carries
`trace_id`/`span_id` next to `request_id`. Telemetry is flushed on graceful shutdown.

---

## 3. Scope

### Included

- `internal/platform/observability`: `Setup(ctx, cfg) (shutdown, error)` building the
  `Resource`, Tracer/Meter providers, the OTLP exporter (no-op without an endpoint), a
  dev stdout exporter, the W3C propagator, and configurable sampling.
- Config: `OTEL_*` fields + parsing; `.env.example`.
- HTTP instrumentation via `otelhttp` (route-named spans + duration/count metrics),
  integrated with the existing middleware chain; low-noise probes.
- Database instrumentation via `otelsql` at the pool (`platform/database`).
- Log/trace correlation: `trace_id`/`span_id` on `slog` records + the request log line.
- The `Tracer()`/`Meter()` seam + documented instrumentation conventions (FR-407).
- `cmd/api/main.go` wiring + shutdown ordering (flush telemetry last).
- Unit tests (in-memory exporter) + gated integration test; CHANGELOG/README/lesson.

### Excluded (later specs / ops — SPEC-004 §2)

- AI-call spans + token/cost metrics → SPEC-005 (via this Meter).
- Market-data ingestion metrics → SPEC-006.
- Feature-specific business spans → each feature spec.
- Hosted backend choice, dashboards, alerting → deploy-time / ops.
- OTel **logs** SDK (keep `slog` + correlation, D4); profiling → future.
- Any `/metrics` scrape endpoint (OTLP push only, D2).

---

## 4. Dependencies

### Technical Dependencies

- SPEC-001 logging/server baseline, SPEC-002 DB pool (`platform/database`), SPEC-003
  middleware chain — all instrumented here.
- **New (first observability deps):**
  - `go.opentelemetry.io/otel` + `…/sdk` (trace + metric)
  - `…/exporters/otlp/otlptrace/otlptracehttp` + `…/otlpmetric/otlpmetrichttp`
    (**OTLP/HTTP**, to avoid the heavy gRPC dependency tree)
  - `…/exporters/stdout/stdouttrace` + `stdoutmetric` (dev)
  - `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` (D3)
  - `github.com/XSAM/otelsql` (D5)

> These are justified by G11 / ADR-0003 (OpenTelemetry is already the chosen approach).
> **Risk:** the OTel modules may require a newer Go toolchain (as pgx did in SPEC-002).
> Pin deliberately and keep the `go` directive controlled; flag any bump for sign-off
> rather than letting it drift (see §8).

### Blocking Decisions (resolved — SPEC-004 §14)

- **D1** Traces + metrics + log-correlation now.
- **D2** OTLP exporter, disabled-by-default, stdout for dev; no `/metrics` endpoint.
- **D3** `otelhttp` for HTTP instrumentation.
- **D4** Keep `slog`, inject `trace_id`/`span_id`.
- **D5** `otelsql` at the pool for DB spans.
- **Implementation pick:** OTLP over **HTTP** (not gRPC) to keep the dependency surface
  lean; span names use the Go 1.23+ `Request.Pattern` (route, low-cardinality).

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/config` | Add `OTEL_*` fields + parsing/validation. |
| `internal/platform/database` | `Connect` opens the pool through `otelsql` so queries are traced. |
| `internal/platform/logging` | A `slog.Handler` (or middleware) adds `trace_id`/`span_id` from context. |
| `internal/transport/http` | Router wrapped with `otelhttp`; request log line gains `trace_id`. |
| `cmd/api/main.go` | `observability.Setup` early; shutdown flush ordered last. |
| `.env.example` / `CHANGELOG.md` / `README.md` / indexes | Updated. |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/platform/observability/observability.go` | `Setup`, `Resource`, providers, OTLP/stdout/no-op exporters, propagator, sampler. |
| `internal/platform/observability/seam.go` | `Tracer(name)` / `Meter(name)` accessors injected into features (FR-407). |
| (test) in-memory span/metric exporter usage | Assert spans/attrs without a network. |

---

## 6. Implementation Strategy

### Approach

Bottom-up, each phase compiling and independently reviewable: config → providers/
exporter → HTTP instrumentation → DB instrumentation → log-correlation + seam → tests
→ docs. Keep the dependency rule intact — OTel lives only in `platform/observability`,
`platform/database`, `platform/logging`, and `transport/http`; **no feature/domain core
imports OTel** (BR-403). No secrets/PII in any attribute (BR-402).

### Rollout Method

**Incremental**, one PR for SPEC-004, reviewed phase-by-phase (the established cadence).

### Rollback Strategy

Greenfield infra, additive and disabled-by-default. Rollback = revert the PR. No data,
no migrations, no schema. With no endpoint configured the feature is inert, so risk to
the running app is minimal even if merged.

---

## 7. Implementation Phases

> The template's domain/persistence/application phases don't map to an infra spec;
> phases below are adapted (as in PLAN-001/002), preserving the bottom-up, build-green
> cadence.

### Phase 1 — Configuration & Dependencies

#### Tasks
- [ ] Extend `Config` with the OTel settings: `OTELExporterEndpoint` (URL; empty ⇒
      disabled), `OTELExporterHeaders` (secret), `OTELServiceName` (default
      `yield-forge`), `OTELTraceSampleRatio` (default 1.0 dev / lower in prod),
      `OTELExporterKind` (`otlp` | `stdout` | `none`, default derived from endpoint).
- [ ] Validate in `Load()` (bad ratio/kind ⇒ fatal, consistent with existing config).
- [ ] Add the OTel modules (OTLP/HTTP, sdk, stdout, `otelhttp`, `otelsql`); `go mod tidy`;
      **verify the `go` directive didn't silently bump** — pin/flag if it did.
- [ ] Update `.env.example` documenting each var (endpoint, headers-as-secret, ratio).

#### Deliverables
- Config loads OTel settings with defaults; deps vendored and pinned; `.env.example`
  current; unit tests cover the new parsing.

---

### Phase 2 — Observability Bootstrap (providers + exporter)

#### Tasks
- [ ] `observability.Setup(ctx, cfg) (func(context.Context) error, error)`: build a
      `Resource` (`service.name`, `service.version` from buildinfo, `deployment.environment`
      = `APP_ENV`), a `TracerProvider` (batch span processor + sampler) and a
      `MeterProvider` (periodic reader).
- [ ] Exporter selection: endpoint set ⇒ OTLP/HTTP (traces + metrics, with headers);
      `stdout` kind ⇒ stdout exporters; otherwise a **no-op** pipeline that never errors
      and never blocks (BR-401).
- [ ] Set the global W3C `TraceContext` + `Baggage` propagator and the global providers.
- [ ] Return a `shutdown` that flushes + closes both providers within a bounded timeout.
- [ ] Wire into `cmd/api/main.go`: `Setup` right after config/logger; `defer shutdown`
      ordered so telemetry is flushed **after** the HTTP server drains and the DB pool
      closes (last out). Export failures are logged, never fatal.

#### Deliverables
- App boots with providers; no endpoint ⇒ inert + no errors; `stdout` ⇒ visible spans;
  graceful shutdown flushes. Unit-tested with an in-memory exporter.

---

### Phase 3 — HTTP Instrumentation (traces + metrics)

#### Tasks
- [ ] Wrap the router with `otelhttp.NewHandler` (outermost, or just inside requestID),
      producing a server span per request + HTTP duration/count metrics.
- [ ] Span/route naming via a `WithSpanNameFormatter` using `Request.Pattern` (Go 1.23+)
      so names are the **route** (`GET /auth/me`), not raw paths — bounded cardinality.
- [ ] Keep `/healthz`/`/readyz` low-noise (filter or drop via a sampler/`WithFilter`).
- [ ] Confirm the existing `requestID`/`logRequests` chain still composes (trace started
      before logging so the log line can read the trace id — Phase 5).

#### Deliverables
- Each request yields a route-named span + metrics; probes don't flood traces;
  unit test asserts a span via an in-memory exporter.

---

### Phase 4 — Database Instrumentation

#### Tasks
- [ ] In `platform/database.Connect`, open the pool through `otelsql` (register the pgx
      driver wrapped, or `otelsql.Open`) with the same DSN + pool settings, so queries
      emit child spans under the request span.
- [ ] Configure `otelsql` to record the operation + **parameterised** statement label
      only — **no argument values** (BR-402); optionally DB metrics.
- [ ] Verify repositories are unchanged (no OTel imports leak into `auth/postgres` etc.).

#### Deliverables
- A DB query under a request shows a child span; no SQL args/PII recorded; feature
  adapters untouched. Exercised by the gated integration test (Phase 6).

---

### Phase 5 — Log/Trace Correlation & the Seam

#### Tasks
- [ ] Add a `slog.Handler` wrapper (or enrich `logRequests`) that, when a span is active
      in `context`, attaches `trace_id`/`span_id` to the record.
- [ ] Add `trace_id` to the per-request `logRequests` line alongside `request_id`.
- [ ] `observability.Tracer(name)` / `Meter(name)` accessors for feature injection;
      document the conventions (low-cardinality span names, no secrets/PII, edge-placed,
      money-valued metrics as `int64`) in a doc comment + README.

#### Deliverables
- Logs emitted within a span carry the trace ids; the seam is available and documented;
  unit test asserts correlation.

---

### Phase 6 — Testing

#### Unit Tests (no backend)
- [ ] `Setup`: no endpoint ⇒ no-op, no error; `stdout`/in-memory ⇒ providers work;
      shutdown flushes.
- [ ] HTTP: a request through the wrapped handler produces a span with the expected
      route name/attributes (in-memory span exporter); probes filtered.
- [ ] Log correlation: a record logged within a span carries `trace_id`/`span_id`.
- [ ] Config: `OTEL_*` parse + defaults + fatal on bad values; propagation continues an
      incoming `traceparent`, else starts a root.

#### Integration Tests (gated by `testing.Short()` + `TEST_DATABASE_URL`)
- [ ] A request that hits the DB yields a parent HTTP span with a child DB span
      (in-memory exporter).
- [ ] Graceful shutdown flushes a pending span.

#### Deliverables
- `go test ./...` green with and without a DB (`-p 1` for the shared-DB integration
  tests, per the SPEC-003 fix); `go vet`/`gofmt` clean; no-PII assertion on attributes.

---

### Phase 7 — Documentation

#### Tasks
- [ ] `CHANGELOG.md` `[Unreleased]`: OTel baseline (traces/metrics/correlation, no-op
      default, new env vars, deps).
- [ ] `README.md`: how to run with a local OTLP collector / Jaeger; how it no-ops
      without one; the new env vars; the instrumentation conventions.
- [ ] Flip SPEC-004 + PLAN-004 to Done; update both indexes.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-004-aula.html`.

#### Deliverables
- Docs current; SPEC-004 closed; lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| OTel modules bump the Go toolchain (as pgx did) | Medium | Pin versions; check the `go` directive after `tidy`; flag any bump for sign-off; prefer OTLP/HTTP to dodge the gRPC tree. |
| Secrets/PII leak into spans/attributes | High | BR-402; `otelsql` records no arg values; review attribute sets; reuse SPEC-002/003 redaction; a test asserting no sensitive keys. |
| Span/label cardinality blowup (raw paths, ids) | Medium | Route-pattern span names (`Request.Pattern`); bounded attributes; probe filtering. |
| Export blocks/slows the request path | High | Batch processor + background export; bounded shutdown flush; failures logged, never on the request path (BR-401). |
| Tracing overhead under load | Low/Med | Configurable sampling (ratio in prod); batch export; measure. |
| Dependency-direction drift (OTel in a feature core) | Medium | Keep OTel in platform/transport only; injected seam; review + build. |
| Noisy background export-failure logs when backend down | Low | Rate-limit / log once; never per-request. |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-401…FR-408 acceptance criteria satisfied.
- [ ] No endpoint ⇒ app runs normally (no-op); endpoint ⇒ traces + metrics exported.
- [ ] HTTP spans route-named; DB child spans present; logs correlated; shutdown flushes.

### Technical Validation
- [ ] OTel confined to `platform/observability`, `platform/database`, `platform/logging`,
      `transport/http`; feature/domain cores import no OTel (BR-403).
- [ ] No secrets/PII in attributes/labels (BR-402); vendor-neutral OTLP only (BR-404).
- [ ] Context carries trace state (BR-405); telemetry never required to run (BR-401).

### Quality Validation
- [ ] Unit tests pass with no backend; integration (in-memory exporter + real DB) passes.
- [ ] `go build`/`go vet`/`gofmt`/`golangci-lint` clean; `go mod tidy`; `go` directive
      reviewed.
- [ ] Code reviewed (hexagonal + go-correctness); CHANGELOG updated in the same PR.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-004 acceptance criteria met.
- [ ] `observability.Setup` + shutdown wired; no-op without a backend; flushes on exit.
- [ ] HTTP route-named spans + request metrics; DB child spans (no args); logs carry
      `trace_id`/`span_id`; the `Tracer`/`Meter` seam documented.
- [ ] New `OTEL_*` config + `.env.example`; tests + lint/vet/fmt clean; `go` directive
      sign-off if changed.
- [ ] `CHANGELOG.md` + `README.md` updated; SPEC-004 + PLAN-004 flipped to Done; indexes
      updated.
- [ ] PR reviewed and merged to `main`.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-004-aula.html` produced.

---

## 11. Deliverables

### Code Deliverables
- `internal/platform/observability/{observability.go,seam.go}`; `otelhttp` wiring in
  `transport/http`; `otelsql` in `platform/database`; `slog` correlation in
  `platform/logging`; `Config` OTel fields; `cmd/api/main.go` Setup + shutdown.

### Infrastructure Deliverables
- `.env.example` `OTEL_*` vars; pinned OTel dependencies in `go.mod`/`go.sum`.

### Documentation Deliverables
- Updated `CHANGELOG.md`, `README.md`, specs/plans indexes; PT-BR lesson HTML.

---

## 12. Post-Implementation Tasks

### Monitoring
- Stand up a local collector / Jaeger to eyeball the first traces; confirm the p95
  latency metric is queryable (validates the SLO instrument).

### Future Improvements
- AI-call spans + token/cost metrics (SPEC-005) and ingestion metrics (SPEC-006) on this
  Meter; a hosted free-tier backend at deploy time; richer DB metrics; continuous
  profiling; possible move to the OTel logs SDK (D4 revisit).

### Technical Debt
- Default sampling ratio is a placeholder until real traffic exists; a dedicated OTel
  ADR may be added if the exporter/backend choice needs recording (ADR-0003 currently
  covers it).
