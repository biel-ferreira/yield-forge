# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | `Insighter` Port & Free/Local LLM Adapter (+ AI guard gates) |
| Feature ID   | SPEC-005 (foundational)                                |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-018, FR-013, FR-014, §6 Principles, §12 LLM strategy |
| Related ADRs | [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md) (free/local LLM), [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) |
| Version      | 0.1.0                                                  |
| Status       | Approved                                               |
| Plan         | PLAN-005 (authored next via /plan-new 005)             |

---

## 2. Overview

### Purpose

Establish the **single seam through which all LLM access flows** — the `Insighter`
port (FR-018) — with one working **free/local** adapter, and wrap it in the two
**binding guard gates** every AI feature must inherit:

- **Explainability gate (FR-013):** no AI output reaches a caller without a structured,
  human-readable explanation.
- **Non-advice gate (FR-014):** no AI output may contain a **transaction order** —
  an imperative buy/sell, a transaction quantity, a price/entry-exit target, or a
  guaranteed-return claim. Naming a sector or asset as a *consideration* for the
  user's own analysis is allowed; every result carries a non-advice disclaimer.

After this spec, the codebase has a provider-swappable `Insighter`, a gate decorator
that makes those guards **structurally unavoidable**, a deterministic fake for
tests/CI, caching + graceful degradation for zero-cost safety, and AI observability —
but **no user-facing insight features yet** (those are SPEC-104+).

### Business Value

- **Turns the product's #1 and #2 binding constraints into code.** "Explainability
  first" and "copilot, never advisor" are enforced *here, once*, so every later AI
  feature (Insights, Rebalancing, Health Score) inherits them and cannot bypass them.
- **De-risks the AI surface.** Provider, prompt strategy, and guardrails are proven on
  a generic task before the real insight engine (SPEC-104) is built on top.
- **Zero cost (ADR-0003).** MVP runs on a **local** model (Ollama, $0, no key) or a
  free hosted tier; paid providers (Claude/OpenAI) are drop-in adapters behind the same
  port — no domain change. Caching + degradation guarantee no silent charges.
- **The multi-agent / MCP seam.** The `Insighter` port is the boundary the future CIO
  multi-agent system and MCP tools plug into (PRD G9/G10) — kept intact from day one.

### Scope

**In scope:** the `Insighter` port (in `internal/insight`); **two adapters — Ollama
(local/dev) and Groq (hosted, OpenAI-compatible)**; the **explainability + non-advice
gate decorator**; the facts-grounded request/response contract; a deterministic **fake**
Insighter; an **in-memory result cache** behind a swappable `Cache` port + throttling +
graceful degradation; configuration; AI observability (latency/model/token-cost,
**never** prompt/PII); tests; CHANGELOG/README/lesson.

**Out of scope (later specs / future):**
- The **Fact Builder** and the concrete insight categories (concentration, allocation,
  market-context) → **SPEC-104** (AI Insight Engine), which *uses* this port.
- Rebalancing assistant, Health Score → SPEC-105 / SPEC-106 (also via this port).
- Any HTTP endpoint exposing insights → the owning feature spec.
- Paid provider adapters (Claude / OpenAI) → drop-in later behind this port; the Groq
  adapter's OpenAI-compatible shape makes OpenAI nearly free to add. Not implemented here.
- Prompt-tuning for specific insight quality → SPEC-104.
- A **persistent (DB) result cache** → deferred behind the `Cache` port; when it's worth
  it, it can reuse SPEC-104's FR-022 history table (avoids a redundant table). SPEC-005
  ships only the in-memory cache.
- Insight **history** (FR-022 — browsable, bounded, clearable) → SPEC-104.

---

## 3. Functional Requirements

> SPEC-005 implements PRD **FR-018** (pluggable LLM) and makes **FR-013 / FR-014** the
> enforcement gates the AI feature specs inherit. It implements no user-facing AI
> *feature* FR directly (those are FR-008…FR-012, owned by SPEC-104+).

### FR-501 — The `Insighter` Port

A single, provider-agnostic port is the only way the application invokes an LLM.

**Acceptance Criteria**
- [ ] `internal/insight` defines `Insighter` with a `ctx`-first method that takes a
      **facts-grounded request** and returns structured, explainable insights or an error.
- [ ] The port lives in the feature core; concrete providers are adapter subpackages
      (`internal/insight/ollama`, …). The core imports no vendor LLM SDK or HTTP type.
- [ ] No code outside an adapter calls an LLM directly — all access is through the port.

### FR-502 — Free/Local + Hosted LLM Adapters

Two working adapters behind the port (D1): **Ollama** (local, $0, no key — dev default,
data stays on-device) and **Groq** (hosted free tier, OpenAI-compatible — the deployed
path).

**Acceptance Criteria**
- [ ] `internal/insight/ollama` talks to a local Ollama (`base URL`, `model`; no key)
      behind the port — the dev default, where portfolio facts never leave the machine.
- [ ] `internal/insight/groq` talks to Groq's **OpenAI-compatible** API (`base URL`,
      `model`, **API key secret**) behind the port — the hosted/deployed path. Built on
      the OpenAI-compatible shape so OpenAI (paid) is a near-free future drop-in.
- [ ] Provider selection is **config only** (`INSIGHTER_PROVIDER=ollama|groq|fake`),
      not a code change (FR-018); each provider's HTTP details are confined to its adapter.
- [ ] Each adapter builds its prompt **from the provided facts** + a non-advice system
      instruction; it never fabricates figures (BR-502).

### FR-503 — Explainability Gate (FR-013)

Every insight that reaches a caller carries a structured explanation; those that don't
are rejected.

**Acceptance Criteria**
- [ ] A gate decorator wraps the `Insighter` and validates that **every** returned
      insight has a non-empty `Explanation`.
- [ ] An output missing an explanation is **rejected** (`ErrMissingExplanation`) before
      reaching the caller — never silently passed through.
- [ ] The gate is applied centrally so any adapter is automatically covered (BR-501).

### FR-504 — Non-Advice Gate (FR-014)

Outputs are validated to exclude **transaction orders**, not asset mentions, and carry a disclaimer.

**Acceptance Criteria**
- [ ] The gate **rejects** only the **order signature** — an imperative buy/sell
      directed at the user (`compre`/`venda`/`buy`/`sell` an asset), a **transaction
      quantity**, a **price/entry-exit target**, or a **guaranteed/certain-return**
      claim (a deterministic validator + the system-prompt design, §14-D3). A ticker
      named as a *candidate for the user's analysis* (e.g. "worth researching," "fits
      your under-weighted logistics sleeve") **passes**. Fail **closed on the order
      signature** (BR-506).
- [ ] Every passed result carries an explicit **non-advice disclaimer**.
- [ ] The validator is unit-tested against a corpus of advice vs non-advice phrasings
      (PT-BR + EN). The corpus **must** include **true-negative** cases — legitimate
      named candidates (e.g. "`HGLG11` is a logistics FII worth analyzing") that must
      **pass** — alongside order phrasings (e.g. "compre 100 `HGLG11`", a price target)
      that must reject.

### FR-505 — Facts-Grounded Contract

The port reasons *over* deterministic facts; it does not invent numbers.

**Acceptance Criteria**
- [ ] The request carries a **structured facts snapshot** (computed upstream; the Fact
      Builder is SPEC-104) plus the task/kind; the adapter prompts from those facts.
- [ ] The contract makes "facts in" explicit; the adapter/system prompt forbids
      inventing figures, and the response references only provided facts (BR-502).
- [ ] Empty/insufficient facts yield a clear error, not a fabricated answer.

### FR-506 — Cost-Safety: Caching, Throttling, Degradation

Free-tier/local limits never cause a silent charge or a hard failure.

**Acceptance Criteria**
- [ ] Results are cached keyed to a **hash of the input facts** (+ task + user) behind a
      swappable `Cache` port. The default is **in-memory** (LRU); identical inputs →
      cached, no LLM call. A persistent (DB) cache is a later drop-in behind the same port (§8).
- [ ] Cache entries carry a TTL; an expired entry is treated as a miss.
- [ ] Provider rate-limit/unavailability **degrades gracefully** —
      `ErrInsightsUnavailable` (a "temporarily unavailable" signal) — and **never retries
      in a way that incurs a charge** (BR-504).
- [ ] A bounded per-call timeout applies; a hung provider can't hang the caller.

### FR-507 — Configuration

Provider selection and credentials are environment-driven (12-factor).

**Acceptance Criteria**
- [ ] Env config: `INSIGHTER_PROVIDER` (`ollama|groq|fake`), per-provider base URL +
      model, the **Groq API key (secret)**, request timeout, and cache TTL. `fake`/disabled
      is the deterministic port (CI/offline) and keeps the app running with "AI off".
- [ ] Secrets only from the env (the Groq key); the Ollama default needs none.
      `.env.example` documents every variable with placeholders.

### FR-508 — AI Observability (no prompt/PII)

Each LLM call is observable end-to-end without leaking sensitive data.

**Acceptance Criteria**
- [ ] A span per `Generate` records model, latency, token usage, **estimated cost
      (`int64` minor units)**, cache hit/miss, and gate outcome — via the SPEC-004 seam.
- [ ] **No prompt text, facts, or PII** is ever put on a span, metric, or log (BR-505);
      portfolio data is sensitive financial information.

### FR-509 — Deterministic Fake & Tests

The port is fully testable offline, with no model and no cost.

**Acceptance Criteria**
- [ ] A `fake` Insighter returns deterministic, explainable, non-advice insights for
      tests/CI and the "AI disabled" mode.
- [ ] Unit tests cover the gates (missing-explanation rejected; advice rejected;
      disclaimer attached), caching (hit/miss), and degradation — all without a network.

### FR-510 — Docs

**Acceptance Criteria**
- [ ] `CHANGELOG.md` + `README.md` updated (the port, how to run the local LLM, env vars,
      the guarantees); `.env.example` updated.
- [ ] On close: SPEC-005 + PLAN-005 → Done, indexes updated, PT-BR lesson produced.

---

## 4. User Flows

> The "user" of SPEC-005 is a **feature developer** (SPEC-104+ consumes this port) and,
> transitively, the investor whose AI output is guaranteed explainable + non-advice.

### Flow 1 — Generate gated insights (happy path)
1. A caller passes a **facts snapshot** + task to `Insighter.Generate`.
2. Cache miss → the adapter prompts the local/free LLM grounded in those facts.
3. The gate validates: each insight has an explanation (FR-013) and no advice (FR-014);
   a disclaimer is attached.
4. The result is cached by facts-hash and returned.

### Flow 2 — Cache hit
1. Same facts as a prior call → the cached, already-gated result is returned with **no
   LLM call** (and no cost).

### Flow 3 — Output violates a guard
1. The LLM returns an insight with no explanation, or containing "buy HGLG11".
2. The gate **rejects** it (`ErrMissingExplanation` / advice rejected) — the caller gets
   an error, never the unsafe output (fail-closed, BR-506).

### Flow 4 — Provider down / rate-limited (degrade)
1. The provider is unreachable or over its free-tier limit.
2. `Generate` returns `ErrInsightsUnavailable`; the caller degrades to a "insights
   temporarily unavailable" state. **No charge, no hang.**

---

## 5. Business Rules (Architectural & Product-Binding)

- **BR-501 — Guards are mandatory middleware.** No path returns AI output without
  passing the explainability + non-advice gates. The gate decorator wraps *every*
  adapter, so a new provider is guarded by construction.
- **BR-502 — Facts are computed, not generated.** The LLM reasons over the deterministic
  facts passed in; adapters never fabricate numbers, and the system prompt forbids it.
  (The Fact Builder that produces the facts is SPEC-104.)
- **BR-503 — Provider behind the port (FR-018).** Vendor SDK/HTTP lives only in adapter
  subpackages; the insight core and all callers depend on the `Insighter` interface.
- **BR-504 — Zero cost / cost-safety.** MVP is free/local; exceeding any free tier
  degrades gracefully and **never charges silently**; caching + throttling bound usage.
- **BR-505 — No prompt/PII in telemetry or logs.** Prompts embed sensitive financial
  data; only model/latency/token/cost/outcome metadata is recorded — never content.
- **BR-506 — Fail closed on the order signature; normalize framing.** A detected
  **order signature** (FR-504 definition) is **rejected** — dropping a real order is
  safer than leaking it — and explainability stays hard fail-closed (no explanation →
  reject). But when the only uncertainty is whether a *named candidate* is phrased as
  a consideration vs. a directive, the gate **normalizes** it to a non-advice
  consideration (and ensures the disclaimer) rather than deleting the insight — so a
  conservative default never silently neuters legitimate portfolio intelligence
  (PRD FR-019).
- **BR-507 — Conventions.** Token **cost is money → `int64` minor units** (CLAUDE.md);
  `ctx` first; identity from context where a user scopes the cache key (SPEC-003).

---

## 6. Domain Model

### Entity: Insight
| Field        | Type    | Notes                                                |
| ------------ | ------- | ---------------------------------------------------- |
| Category     | string  | e.g. concentration / allocation / market (refined in SPEC-104) |
| Title        | string  | short headline                                       |
| Detail       | string  | the observation, framed as a *consideration*; may name a sector or candidate asset, never a transaction order |
| Explanation  | string  | **required** — why this was raised (FR-013)          |

### Value object: InsightRequest
| Field   | Type                 | Notes                                              |
| ------- | -------------------- | -------------------------------------------------- |
| Facts   | structured snapshot  | deterministic, computed upstream (BR-502)          |
| Task    | enum/kind            | what to reason about (generic in SPEC-005)         |
| UserID  | string (from ctx)    | scopes the cache key (SPEC-003 BR-304)             |

### Value object: InsightResult
| Field      | Type        | Notes                                            |
| ---------- | ----------- | ------------------------------------------------ |
| Insights   | []Insight   | each gated (explained + advice-free)             |
| Disclaimer | string      | non-advice disclaimer, always present (FR-014)   |

Ports (in `internal/insight`): `Insighter` (the LLM seam), `Cache` (result cache).
Sentinel errors: `ErrMissingExplanation`, `ErrAdviceDetected`, `ErrInsightsUnavailable`,
`ErrInsufficientFacts`.

---

## 7. API Specification

**No new HTTP endpoints.** SPEC-005 is an internal port + adapter + gates; the AI
*features* that call it (SPEC-104+) own their endpoints and DTOs. The existing API is
unchanged. The "contract" here is the Go `Insighter` interface and the gated
`InsightResult` shape.

---

## 8. Data Storage

**None — no migration.** Caching ships **in-memory** behind a swappable `Cache` port.
For a single-user MVP on free providers, this captures essentially all the benefit: the
cache saves **latency / rate-limit budget** (not money — both providers are free), and
an in-memory cache covers a stable portfolio's repeated views with zero infra.

A **persistent (DB) cache** is deferred behind the same port. When it's worth it (a
restart-happy free host pressuring Groq's daily limit), it should **reuse SPEC-104's
FR-022 history table** — a recent history row with a matching facts-hash within TTL is a
cache hit — rather than a separate `insight_cache` table that would duplicate it. No
feature tables here (insight *history* is SPEC-104).

---

## 9. Edge Cases

| Scenario | Expected behaviour |
| -------- | ------------------ |
| LLM output missing an explanation | Rejected (`ErrMissingExplanation`); caller never sees it. |
| LLM output contains an order — "compre/venda <ticker>", a quantity, or a price target | Rejected (order signature); fail closed (BR-506). |
| LLM output names a ticker as a *consideration* ("HGLG11 worth analyzing") | **Passes** (not an order); disclaimer attached (FR-504). |
| Malformed/non-JSON LLM output | One bounded re-ask, then degrade (`ErrInsightsUnavailable`). |
| Provider unreachable / timeout | Degrade gracefully; bounded timeout; no hang, no charge. |
| Provider rate-limited (free tier) | Degrade; do not retry in a way that charges (BR-504). |
| Empty/insufficient facts | `ErrInsufficientFacts` — no fabricated answer (BR-502). |
| Identical facts repeated | Cache hit; no LLM call, no cost. |
| AI disabled (config) | Port is the deterministic fake / clear "AI off"; app still runs. |

---

## 10. Security Considerations

- **Prompts carry sensitive financial PII** → never logged, never on spans/metrics
  (BR-505); only metadata (model/latency/tokens/cost/outcome).
- **API keys** (hosted providers) are secrets from the env; `.env.example` placeholders;
  never committed. The local default needs no key.
- **TLS** to any hosted provider; the local model is loopback only.
- **Prompt-injection awareness:** facts are structured/escaped, not free user text, and
  the output gates are the backstop — a model coaxed into "advice" is still rejected by
  the non-advice gate (defense in depth).
- **Output is never executable advice** — the non-advice gate is a product-safety
  control, not just a UX nicety (FR-014).

---

## 11. Observability

- **Spans/metrics (via SPEC-004 seam):** per `Generate` — model, latency, token usage,
  estimated cost (`int64`), cache hit/miss, and gate outcome (passed / rejected-no-expl /
  rejected-advice / degraded). Enables the PRD's "AI call latency & token usage/cost"
  and "insight generation success rate" metrics (§10).
- **No content:** prompts, facts, and generated text are never emitted (BR-505).
- **Logs:** a gate rejection logs at `warn` with the *reason code* only (no content).

---

## 12. Testing Strategy

### Unit Tests
- **Gates:** explanation-missing → rejected; **order** phrasings (imperative
  "compre/venda" + asset, transaction quantity, price/entry-exit target, guaranteed
  return) → rejected; **named candidates as considerations** (e.g. "`HGLG11` is a
  logistics FII worth analyzing") → **pass**; clean output → passes with a disclaimer
  attached. Table-driven, PT-BR + EN corpus with explicit **true-negative** cases.
- **Cache:** miss calls the underlying Insighter once; hit returns cached without calling
  it; key changes with facts.
- **Degradation:** a fake adapter returning a provider error → `ErrInsightsUnavailable`,
  no panic, bounded.
- **Fake Insighter:** deterministic, explainable, non-advice output.

### Integration Tests (optional, gated)
- Against a **local Ollama** if present (env-gated like `TEST_DATABASE_URL`): a real
  generation passes the gates end-to-end. Skips cleanly when no model is available (CI
  has none) — the deterministic fake covers CI.

### Quality gate
- `go build`/`go vet`/`gofmt` clean; unit tests pass with no model/network; the insight
  core imports no vendor LLM SDK/HTTP (BR-503); no content in telemetry asserted.

---

## 13. Definition of Done

- [ ] `internal/insight` with the `Insighter` + `Cache` ports, domain types, and sentinels.
- [ ] **Ollama + Groq** adapters behind the port (+ deterministic `fake`); provider
      swappable by config (`INSIGHTER_PROVIDER=ollama|groq|fake`).
- [ ] The **explainability + non-advice gate decorator**, applied centrally (BR-501),
      fail-closed on the order signature (BR-506), disclaimer attached.
- [ ] Facts-grounded request contract; adapter prompts from facts; no fabricated numbers.
- [ ] **In-memory** `Cache` (facts-hash key, TTL) behind a swappable port + graceful
      degradation + bounded timeout.
- [ ] Config (provider/model/url/Groq key/timeout/TTL) env-driven; `.env.example` updated.
- [ ] AI spans/metrics via the SPEC-004 seam, with **no prompt/PII**.
- [ ] Deterministic fake; unit tests (gates, cache, degradation) pass with no network;
      optional gated Ollama integration test.
- [ ] `CHANGELOG.md` + `README.md` updated; SPEC-005 + PLAN-005 → Done; indexes updated;
      PT-BR lesson produced.
- [ ] PLAN-005 followed; PR reviewed (hexagonal + go-correctness) and merged.

---

## 14. Decisions (resolved)

> Confirmed with the project owner before PLAN-005. These are now binding.

- **D1 — Two adapters: Ollama (local/dev) + Groq (hosted/deployed), plus a deterministic
  fake for CI.** ✅ Ollama is the dev default ($0, no key, portfolio data stays
  on-device); **Groq** is the deployed path (generous free tier, **OpenAI-compatible**
  API — so OpenAI/other compatible providers are a near-free future drop-in). Provider is
  a config swap (`INSIGHTER_PROVIDER=ollama|groq|fake`). Gemini was the runner-up; Claude/
  OpenAI remain future paid adapters behind the same port.
- **D2 — Structured JSON response (each insight has an `explanation` field), parsed +
  validated.** ✅ Makes the explainability gate deterministic; one bounded re-ask on
  malformed output. Free-text + heuristics rejected.
- **D3 — Deterministic non-advice validator targeting the *order signature*** — imperative
  buy/sell + asset, transaction quantity, price/entry-exit target, guaranteed return —
  **that rejects, plus a non-advice system prompt.** ✅ Belt-and-suspenders; fail closed on
  the order signature. A ticker named as a *consideration* must **pass** (true-negative
  corpus, FR-504 / PRD FR-019). Prompt-only and sanitize-in-place rejected.
- **D4 — In-memory result cache** behind a swappable `Cache` port. ✅ For a single-user
  MVP on free providers the cache saves latency/rate-limit (not money), and in-memory
  captures ~all of that with zero infra. A persistent (DB) cache is deferred behind the
  same port and, when needed, should **reuse SPEC-104's FR-022 history table** rather
  than a redundant `insight_cache` table. (Reverted from a brief DB-cache decision once
  the cache-vs-history overlap was clear.)
- **D5 — Gate decorator wrapping any `Insighter`.** ✅ `gated(insighter)` — guards are
  structural, provider-agnostic, impossible to forget; composes with the cache decorator.
  Gates-inside-each-adapter rejected.

---

## 15. Open Questions (deferred, not blocking)

- Specific Groq model id (e.g. a Llama variant) — picked during PLAN-005; the adapter is
  model-agnostic via config.
- Cache invalidation policy beyond TTL (e.g. evict on portfolio change) — refined with
  SPEC-104's Fact Builder; TTL + facts-hash key suffice for the MVP.
- Token-cost estimation table per provider/model (for the cost metric) — refined when a
  paid provider is wired; local cost is 0.
- Prompt templates per insight category + the Fact Builder — **SPEC-104**.
- Streaming responses (for a future UI) — not needed for the MVP batch generation.
- Multi-agent CIO orchestration over this port (PRD G9) — future; the port is the seam.
