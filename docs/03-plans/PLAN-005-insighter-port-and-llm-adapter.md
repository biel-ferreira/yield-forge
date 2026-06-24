# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | `Insighter` Port & Free/Local LLM Adapter (+ AI guard gates) |
| Related Feature | Foundational — the AI seam + binding-guard gates             |
| Related Spec    | [SPEC-005](../02-specs/SPEC-005-insighter-port-and-llm-adapter.md) |
| Version         | 0.1.0                                                        |
| Status          | Done                                                         |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-23                                                   |

---

## 2. Objective

### Goal

Build the single seam through which all LLM access flows — the `Insighter` port — with
two adapters (**Ollama** local/dev, **Groq** hosted/deployed), and wrap it in the two
**binding-guard gates** (explainability FR-013, non-advice FR-014) so every later AI
feature inherits them and cannot bypass them. Add an in-memory result cache, graceful
degradation, and AI observability — with **no user-facing AI feature yet** (SPEC-104+).

### Expected Outcome

A caller can pass a facts snapshot to `Insighter.Generate` and get back structured,
**explained, advice-free** insights (+ disclaimer) — or a clean degradation error if the
provider is down. Provider is a config swap (`ollama|groq|fake`). Identical facts hit the
cache (no LLM call). Each call emits an AI span (model/latency/tokens/cost, cache hit,
gate outcome) with **no prompt/PII**. The gates are a decorator, structurally
unavoidable. CI runs fully on the deterministic fake (no model, no network).

---

## 3. Scope

### Included

- `internal/insight`: domain (`Insight`/`InsightRequest`/`InsightResult`), the
  `Insighter` + `Cache` ports, sentinel errors, the non-advice validator, the **gate
  decorator**, the **in-memory cache** decorator, the deterministic **fake**, and a
  `factory` that composes them from config.
- Adapters: `internal/insight/ollama` (local) and `internal/insight/groq`
  (OpenAI-compatible), both over **stdlib `net/http`** (no vendor SDK).
- Config (`INSIGHTER_*`) + `.env.example`; AI observability via the SPEC-004 seam.
- Unit tests (gate corpus PT-BR+EN, cache, degradation, fake); optional gated Ollama
  integration test; CHANGELOG/README/lesson.

### Excluded (later specs / future — SPEC-005 §2)

- The **Fact Builder** + concrete insight categories, prompt-tuning, and any HTTP
  endpoint → **SPEC-104**.
- The FR-022 **history** feature (browse/clear, bounded memory) → SPEC-104.
- A **persistent (DB) cache** → deferred behind the `Cache` port; would reuse SPEC-104's
  history table (no migration here).
- Paid adapters (Claude/OpenAI) → drop-in later (Groq's OpenAI-compatible shape makes
  OpenAI nearly free to add).
- The FR-020 **risk/assumption disclosure** check on *suggestions* → extends the same
  gate when suggestions land (SPEC-105); the gate is built to leave that seam open.

---

## 4. Dependencies

### Technical Dependencies

- SPEC-004 observability seam (`observability.Tracer()/Meter()`) — for AI spans/metrics.
- SPEC-003 `auth.UserID(ctx)` — scopes the cache key per user (BR-304).
- A local **Ollama** (optional, dev only) for the live adapter / gated integration test.
- A **Groq API key** (optional, for the hosted adapter / manual verification).

### New Dependencies

- **None (stdlib-first).** Both adapters use `net/http` + `encoding/json`; the cache uses
  `container/list` (or a tiny LRU) + `sync`; facts-hash uses `crypto/sha256`. Groq's
  OpenAI-compatible API is plain JSON — no SDK needed (ADR-0003 lean posture).

### Blocking Decisions (resolved — SPEC-005 §14)

- **D1** Ollama (dev) + Groq (deployed, OpenAI-compatible) + deterministic fake.
- **D2** Structured JSON response (each insight has `explanation`); one bounded re-ask.
- **D3** Deterministic non-advice validator on the **order signature** (imperative
  buy/sell + asset, quantity, price/entry-exit target, guaranteed return) — rejects;
  named candidates as considerations **pass** (true-negative corpus). Fail closed on the
  order signature.
- **D4** **In-memory** cache behind a swappable `Cache` port (no migration).
- **D5** Guard **decorator** wrapping any `Insighter`.
- **Default provider:** `INSIGHTER_PROVIDER` defaults to `ollama` (dev intent); CI/tests
  use `fake`; deployed uses `groq`. Nothing calls `Generate` until SPEC-104, so the
  default is inert for now.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/config` | Add `INSIGHTER_*` fields + validation. |
| `.env.example` / `CHANGELOG.md` / `README.md` / indexes | Updated. |
| `internal/insight/doc.go` | Replaced by real package contents (was a placeholder). |
| `cmd/api/main.go` | **No wiring yet** — no consumer until SPEC-104; the factory is unit-tested. (Optionally validate config at startup; not required.) |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/insight/insight.go` | Domain types + sentinel errors. |
| `internal/insight/ports.go` | `Insighter` + `Cache` interfaces. |
| `internal/insight/advice.go` | Non-advice validator (order-signature detection). |
| `internal/insight/gate.go` | Gate decorator (explainability + non-advice, disclaimer). |
| `internal/insight/cache.go` | In-memory LRU+TTL cache decorator + facts-hash key. |
| `internal/insight/fake.go` | Deterministic, explainable, non-advice Insighter. |
| `internal/insight/prompt.go` | Shared facts→prompt builder + non-advice system prompt. |
| `internal/insight/factory.go` | Compose `observed(cached(gated(adapter)))` from config. |
| `internal/insight/ollama/` | Ollama adapter (HTTP, JSON mode). |
| `internal/insight/groq/` | Groq adapter (OpenAI-compatible chat completions). |

---

## 6. Implementation Strategy

### Approach

Bottom-up, each phase compiling and independently reviewable: domain/ports → **gates**
(the binding constraints, built and tested *before* any real adapter) → Ollama → Groq →
cache/degradation/observability/factory → tests → docs. The composition is
`observed(cached(gated(provider)))`: the cache stores **already-gated** results, so a hit
is safe; observability is outermost (records cache hit/miss + gate outcome + adapter
metrics). The insight **core imports no vendor LLM SDK / HTTP** (BR-503) — those live in
the adapter subpackages and `prompt.go` builds provider-neutral messages.

### Rollout Method

**Incremental**, one PR for SPEC-005, reviewed phase-by-phase (established cadence).
Security-sensitive (AI guardrails) → suggest `/security-review` before close.

### Rollback Strategy

Additive infra, no migration, no consumer yet — rollback = revert the PR. With the
default provider inert (nothing calls `Generate` until SPEC-104), risk to the running app
is near zero.

---

## 7. Implementation Phases

> Adapted from the layered order (no persistence/API phases — in-memory cache, no HTTP).
> The **gates** are pulled early (Phase 2) because they are the binding product guards
> and everything else composes around them.

### Phase 1 — Domain, Ports & Config

#### Tasks
- [ ] `internal/insight`: `Insight{Category,Title,Detail,Explanation}`,
      `InsightRequest{Facts,Task,UserID}`, `InsightResult{Insights,Disclaimer}`; sentinels
      (`ErrMissingExplanation`, `ErrAdviceDetected`, `ErrInsightsUnavailable`,
      `ErrInsufficientFacts`).
- [ ] `ports.go`: `Insighter.Generate(ctx, InsightRequest) (InsightResult, error)` and
      `Cache.Get/Set` (consumer-defined; ctx-first).
- [ ] Extend `Config` with `INSIGHTER_*` (provider, ollama url/model, groq url/model/key,
      timeout, cache TTL + size); validate (bad provider/timeout fatal; groq key required
      iff provider=groq). `.env.example` updated.

#### Deliverables
- `insight` core compiles (pure domain + ports); config loads/validates; unit tests for
  validation + config parsing.

---

### Phase 2 — The Guard Gates (binding constraints)

#### Tasks
- [ ] `advice.go`: a deterministic validator detecting the **order signature** —
      imperative buy/sell + asset (`compre`/`venda`/`buy`/`sell`), transaction quantity,
      price/entry-exit target, guaranteed/certain-return claim. B3 ticker shapes
      (e.g. `HGLG11`) recognized but a ticker named as a *consideration* must **pass**.
- [ ] `gate.go`: a decorator wrapping any `Insighter` that, on every result, (a) rejects
      if any insight lacks an `Explanation` (`ErrMissingExplanation`); (b) rejects on a
      detected order signature (`ErrAdviceDetected`); (c) attaches the non-advice
      disclaimer. Fail **closed** (BR-506). Structured so the FR-020 risk-disclosure check
      is a clean future addition.
- [ ] Log a gate rejection at `warn` with the **reason code only** (no content, BR-505).

#### Deliverables
- The gate decorator + validator, **unit-tested against a PT-BR+EN corpus** including
  explicit **true-negatives** (named candidates that must pass) and order phrasings that
  must reject. This is the spec's centerpiece — reviewed hardest.

---

### Phase 3 — Ollama Adapter (local/dev)

#### Tasks
- [ ] `prompt.go`: build provider-neutral chat messages from `InsightRequest.Facts` + a
      strict **non-advice + facts-grounded system prompt** (forbid inventing numbers /
      issuing orders); request **JSON output**.
- [ ] `internal/insight/ollama`: `net/http` client to `/api/chat` (JSON `format`), bounded
      by `ctx`/timeout; parse the structured JSON into `InsightResult`; one bounded re-ask
      on malformed JSON, then `ErrInsightsUnavailable`. Map unreachable/timeouts to
      `ErrInsightsUnavailable` (degrade, never hang).

#### Deliverables
- A working local adapter behind the port; provider HTTP confined to the subpackage;
  unit-tested with an `httptest` server (success, malformed→re-ask, error→degrade).

---

### Phase 4 — Groq Adapter (hosted, OpenAI-compatible)

#### Tasks
- [ ] `internal/insight/groq`: `net/http` client to the **OpenAI-compatible**
      `/chat/completions` (Bearer API key, JSON response format, model from config),
      reusing `prompt.go`. Same parse + re-ask + degrade behaviour.
- [ ] Map auth/rate-limit (429) and 5xx to `ErrInsightsUnavailable` (degrade; **no
      charge-incurring retry**, BR-504); never log the key or prompt.

#### Deliverables
- A hosted adapter behind the same port; OpenAI-compatible shape (so OpenAI is a future
  drop-in); unit-tested with an `httptest` server (success, 429→degrade, 401→degrade).

---

### Phase 5 — Cache, Degradation, Observability & Factory

#### Tasks
- [ ] `cache.go`: an in-memory **LRU + TTL** `Cache` impl; key = `sha256(facts + task +
      userID)`; a cache **decorator** that wraps a gated `Insighter` (stores **gated**
      results). Cache hit → no LLM call.
- [ ] `factory.go`: `New(cfg, tracer, meter) Insighter` composing
      `observed(cached(gated(provider)))`, selecting the provider from config (`fake` for
      CI/disabled).
- [ ] AI observability via the SPEC-004 seam: a span per `Generate` with model, latency,
      token usage, **estimated cost (`int64` minor units)**, cache hit/miss, and gate
      outcome — and **no prompt/facts/content** (BR-505). Cost is `int64` (CLAUDE.md);
      free providers report 0.

#### Deliverables
- End-to-end (with the fake): facts → gated, cached, observed result; degradation path
  returns `ErrInsightsUnavailable`; a span carries metadata only. Unit-tested.

---

### Phase 6 — Testing

#### Unit Tests (no model/network)
- [ ] **Gates** (corpus): missing-explanation → reject; order phrasings (PT-BR "compre
      100 HGLG11", price target, guaranteed return) → reject; **named candidates as
      considerations** → pass; clean → pass + disclaimer.
- [ ] **Adapters** (`httptest`): success parse; malformed → one re-ask → degrade;
      provider error / 429 / 401 → `ErrInsightsUnavailable`.
- [ ] **Cache**: miss calls underlying once; hit returns cached without calling; key
      changes with facts; TTL expiry = miss.
- [ ] **Fake**: deterministic, explainable, non-advice output.
- [ ] **No content in telemetry**: assert spans carry no prompt/facts/PII.

#### Integration Tests (optional, gated)
- [ ] Against a local **Ollama** if present (env-gated, skips cleanly in CI): a real
      generation passes the gates end-to-end. The fake covers CI.

#### Deliverables
- `go test ./... -short` green with no model/network; `go vet`/`gofmt` clean; the insight
  core imports no vendor LLM SDK/HTTP (BR-503).

---

### Phase 7 — Documentation

#### Tasks
- [ ] `CHANGELOG.md` `[Unreleased]`: the `Insighter` port, the two adapters, the guard
      gates, in-memory cache, AI observability, new env vars.
- [ ] `README.md`: the port + the two guarantees (explainable, non-advice), how to run
      the local Ollama, the `INSIGHTER_*` env vars, and the provider swap.
- [ ] Flip SPEC-005 + PLAN-005 to Done; update both indexes.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-005-aula.html`.

#### Deliverables
- Docs current; SPEC-005 closed; lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Non-advice validator false-positive (eats legitimate candidates) or false-negative (lets an order through) | High | Order-signature-only matching (not bare action+ticker proximity); a large PT-BR+EN corpus with **true-negatives**; fail closed only on the order signature; reviewed hardest (and `/security-review`). |
| Prompt/facts/PII leaking into telemetry or logs | High | BR-505; record metadata only; a test asserts no content on spans; gate logs the reason code only. |
| LLM returns non-JSON / hallucinated numbers | Medium | JSON-mode request + defensive parse + one bounded re-ask, then degrade; facts-grounded system prompt forbids inventing figures; the gate is the backstop. |
| Provider hangs / rate-limits / costs | Medium | Bounded per-call timeout + ctx; map errors to `ErrInsightsUnavailable`; no charge-incurring retry (BR-504); free providers only at MVP. |
| Vendor HTTP/SDK leaking into the insight core | Medium | `net/http` confined to adapter subpackages; core imports only ports; review + build. |
| Ollama unavailable in CI | Low | Deterministic fake covers CI; Ollama integration test is gated/skips. |
| Token-cost estimation accuracy | Low | Free providers report 0; a per-model cost table is deferred (§15) — the metric shape (`int64`) is fixed now. |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-501…FR-510 acceptance criteria satisfied.
- [ ] Gates: no output without an explanation; no order signature passes; disclaimer always present.
- [ ] Provider is a config swap (ollama|groq|fake); cache hit avoids the LLM call;
      degradation returns `ErrInsightsUnavailable` without hang/charge.

### Technical Validation
- [ ] Insight core imports no vendor LLM SDK/HTTP (BR-503); gates are a central decorator (BR-501).
- [ ] No prompt/facts/PII in telemetry or logs (BR-505); token cost is `int64` (BR-507).
- [ ] Cache key scoped by `userID` from context (BR-304); facts-grounded prompt (BR-502).

### Quality Validation
- [ ] Unit tests pass with no model/network; optional gated Ollama integration passes locally.
- [ ] `go build`/`go vet`/`gofmt`/`golangci-lint` clean; `go mod tidy` (expected: no new deps).
- [ ] Reviewed (hexagonal + go-correctness); `/security-review` for the guard gates;
      CHANGELOG updated in the same PR.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-005 acceptance criteria met.
- [ ] `Insighter` + `Cache` ports, domain, sentinels; the **gate decorator** (fail-closed)
      with the non-advice validator + disclaimer.
- [ ] **Ollama + Groq** adapters (stdlib HTTP) + deterministic **fake**; provider via config.
- [ ] In-memory cache (facts-hash key, TTL) + graceful degradation + bounded timeout.
- [ ] AI spans/metrics via the SPEC-004 seam, **no prompt/PII**; cost `int64`.
- [ ] Config + `.env.example`; tests green with no network; build/vet/fmt clean.
- [ ] `CHANGELOG.md` + `README.md` updated; SPEC-005 + PLAN-005 → Done; indexes updated.
- [ ] PR reviewed (hexagonal + go-correctness + security) and merged.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-005-aula.html` produced.

---

## 11. Deliverables

### Code Deliverables
- `internal/insight/*` (domain, ports, gate, advice, cache, fake, prompt, factory),
  `internal/insight/ollama`, `internal/insight/groq`; `Config` `INSIGHTER_*` fields.

### Infrastructure Deliverables
- `.env.example` `INSIGHTER_*` vars. (No migration — in-memory cache.)

### Documentation Deliverables
- Updated `CHANGELOG.md`, `README.md`, specs/plans indexes; PT-BR lesson HTML.

---

## 12. Post-Implementation Tasks

### Monitoring
- Confirm AI spans appear (model/latency/tokens/cost, cache hit, gate outcome) with no
  content; watch the insight-generation success-rate / gate-rejection metrics (PRD §10).

### Future Improvements
- FR-020 risk/assumption disclosure check on suggestions (SPEC-105) via the same gate.
- Persistent cache reusing SPEC-104's FR-022 history table; per-model token-cost table;
  Claude/OpenAI paid adapters (OpenAI ≈ free via the Groq adapter shape); streaming.

### Technical Debt
- Default Groq model id is a placeholder until tuned; the non-advice corpus grows as real
  outputs surface edge phrasings.
