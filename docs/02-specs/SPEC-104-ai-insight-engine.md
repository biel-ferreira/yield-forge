# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | AI Insight Engine (Fact Builder + explainable insights) |
| Feature ID   | SPEC-104 (feature)                                     |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-008, FR-009, FR-010, FR-013, FR-014, FR-019/020/021, Epic 5, §6 Principles |
| Related ADRs | [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md), [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) |
| Version      | 0.1.0                                                  |
| Status       | Done                                                   |
| Plan         | [PLAN-104](../03-plans/PLAN-104-ai-insight-engine.md)  |

---

## 2. Overview

### Purpose

Generate **explainable insights** about the investor's portfolio (FR-008/009/010) by
assembling deterministic **facts** — from the profile (SPEC-101), the dashboard computation
(SPEC-103), the holdings (SPEC-102), and macro data (SPEC-006) — and handing them to the
**`Insighter` port** (SPEC-005), whose gates already enforce explainability (FR-013) and
non-advice (FR-014). This is the feature that finally turns the whole stack into the product's
core promise: *spot concentration, imbalance, and risk without manual analysis.*

### Business Value

Every prior spec exists to make this trustworthy: the facts are computed (not invented), the
LLM only reasons over them, and the binding guards make the output safe by construction. The
engine is the seam where the **Fact Builder + CIO multi-agent** vision (PRD §6) takes shape.

### Scope

**In scope**

- A **Fact Builder** that deterministically composes `insight.Facts` from the read seams
  (dashboard, profile, macro) — exposed as a **reusable, published seam** (`BuildFacts`), the
  grounding source the Conversational Copilot (SPEC-108) and SPEC-105/106 consume.
- An insight **engine service** that builds facts and calls the `Insighter` once per category
  — **Portfolio**, **Allocation**, **Market Context** — aggregating the gated results.
- HTTP `GET /insights` (per-user, auth-protected), graceful degradation, observability, tests.
- Reuse of the SPEC-005 Insighter chain (gates + cache + degradation + AI telemetry) — **not**
  re-implemented here.

**Out of scope**

- The Rebalancing Assistant (FR-011, SPEC-105), the Health Score (FR-012, SPEC-106), and the
  projections (SPEC-107).
- **Insight history / persistence (FR-022)** — flagged as a decision (D4); recommended as a
  focused follow-up, not part of this spec.
- Any new LLM provider/adapter or change to the gates (those are SPEC-005); any new market-data
  source (SPEC-006).

---

## 3. Functional Requirements

### FR-1041 — Fact Builder (facts are computed, not generated)

#### Acceptance Criteria

- [ ] Deterministically assembles a fact set for the authenticated user from: the dashboard
      (current value, allocation by class, FII sector exposure, largest-holding concentration,
      stale tickers — SPEC-103), the profile (risk, objectives, horizon — SPEC-101), and the
      latest macro indicators (SELIC, CDI, IPCA — SPEC-006).
- [ ] All money in the facts is `int64` centavos and all rates/shares integer **basis points**
      — never float; the same inputs always produce the same facts (PRD §6).
- [ ] The facts carry only computed values — the engine never lets the LLM invent numbers.

### FR-1042 — Portfolio Insights (FR-008)

#### Acceptance Criteria

- [ ] Produces insights on **concentration, sector imbalance, risk exposure, and
      diversification**, grounded in the facts.

### FR-1043 — Allocation Insights (FR-009)

#### Acceptance Criteria

- [ ] Produces insights on **allocation alignment** (asset-class mix vs the investor's risk
      profile/objectives) and **single-sector / single-asset concentration**.

### FR-1044 — Market Context Insights (FR-010)

#### Acceptance Criteria

- [ ] Produces insights tying **macro conditions** (e.g. a high-SELIC environment, inflation)
      to the portfolio's FI/FII split — framed as considerations, not directives.

### FR-1045 — Explainability, by construction (FR-013)

#### Acceptance Criteria

- [ ] **Every** insight reaching the user carries a human-readable explanation — guaranteed by
      the Insighter's explainability gate (SPEC-005). The engine surfaces only gated output and
      never constructs user-facing AI text outside the Insighter.

### FR-1046 — Non-Advice, by construction (FR-014)

#### Acceptance Criteria

- [ ] No insight contains a transaction order, quantity, price/entry-exit target, imperative
      buy/sell, or guaranteed return — rejected by the Insighter's non-advice gate; the
      response carries the non-advice disclaimer. Naming a sector/asset as a *consideration* is
      allowed (FR-019/FR-014).

### FR-1047 — Graceful Degradation & Empty State

#### Acceptance Criteria

- [ ] If the LLM is unavailable, the engine degrades to a clear "insights temporarily
      unavailable" state (the dashboard/portfolio remain usable); a partial result is returned
      when only some categories fail.
- [ ] An **empty portfolio** returns a friendly "add holdings to get insights" state — no LLM
      call, not an error.

### FR-1048 — API

#### Acceptance Criteria

- [ ] `GET /insights` returns the caller's aggregated insights (each tagged with its category)
      plus the disclaimer; per-user, behind the deny-by-default auth middleware; registered in
      the `routeTable` and documented in `api/openapi.yaml` (drift test).

### FR-1049 — Observability

#### Acceptance Criteria

- [ ] The endpoint inherits the `otelhttp` route span; the Insighter records its AI spans
      (provider/model/outcome/cache-hit, **no prompt/facts/PII** — SPEC-005 BR-505); the
      Fact Builder adds a span with **no PII**.

### FR-1050 — Documentation

#### Acceptance Criteria

- [ ] `README` + `CHANGELOG` updated; the PT-BR lesson `docs/lessons/SPEC-104-aula.html`
      produced on close.

---

## 4. User Flows

### Main Flow

1. The authenticated user `GET /insights`.
2. The Fact Builder composes the deterministic fact set (dashboard + profile + macro).
3. The engine calls the `Insighter` once per category (portfolio, allocation, market context),
   each request carrying the facts + a category task; the Insighter gates each result.
4. The engine aggregates the gated insights (tagged by category) + the disclaimer and returns them.

### Alternative Flows

- **Empty portfolio** → a friendly empty state, no LLM call.
- **LLM down** → "insights temporarily unavailable" (or a partial result if some categories
  succeeded).

---

## 5. Business Rules

- **BR-1041 — Facts are computed, not generated.** The Fact Builder produces deterministic
  facts; the LLM reasons over them and never invents numbers (the binding constraint, PRD §6).
- **BR-1042 — Guards by construction.** Every user-facing AI string passes the SPEC-005
  explainability + non-advice gates because the engine emits output *only* through the
  `Insighter`; it never bypasses the port. (FR-013/FR-014 are enforced upstream, not re-coded.)
- **BR-1043 — Identity from context.** Facts are built for the `auth.UserID(ctx)` only;
  per-user scoping flows through the read seams (no cross-user data, no client-supplied id).
- **BR-1044 — Money in facts is `int64` centavos / integer bps**, never float — consistent
  with the dashboard it reuses (BR-1032).
- **BR-1045 — Zero cost / degrade.** Caching, throttling, cost-safety, and graceful
  degradation live in the Insighter chain (SPEC-005); the engine surfaces the degraded state,
  never a hard failure for an LLM outage.
- **BR-1046 — Read-only over features (acyclic).** The engine composes the dashboard, profile,
  and macro read seams plus the Insighter port; those features do not depend on insight.
- **BR-1047 — Conventions.** Errors `%w` + sentinels; `ctx` first; DTOs separate from domain;
  no package-name stutter; doc comments cite SPEC/BR.

---

## 6. Domain Model

### Value object: Category

Closed enum mapping to an `insight.Task`: `portfolio` | `allocation` | `market_context`.

### Computed / produced

| Type | Notes |
| ---- | ----- |
| `insight.Facts` | `map[string]any` of computed facts (money centavos / rates bps) — built by the Fact Builder. |
| `insight.Insight` | `{ Category, Title, Detail, Explanation }` — produced by the Insighter (SPEC-005). |
| `Insights` (engine result) | `{ Items []insight.Insight; Disclaimer string; Available bool }` — the aggregate across categories. |

Reuses `dashboard.Dashboard`, `profile.Profile`, `marketdata.MacroIndicator`, and the
`insight.Insighter`/`Facts`/`InsightResult` from SPEC-005.

---

## 7. Ports

### Input ports (consumer-defined — what the engine reads)

```go
// The engine reads facts through small consumer interfaces (accept interfaces), satisfied at
// the wiring edge by the dashboard service, the profile service, and the macro repository.
type DashboardReader interface {
    GetDashboard(ctx context.Context, userID string) (dashboard.Dashboard, error)
}
type ProfileReader interface {
    GetProfile(ctx context.Context, userID string) (profile.Profile, error) // profile.ErrProfileNotFound when absent
}
type MacroReader interface {
    GetLatestMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error)
}

// The Insighter port (SPEC-005) is the only path to user-facing AI text — gates included.
//   insight.Insighter.Generate(ctx, insight.InsightRequest{Facts, Task, UserID}) (insight.InsightResult, error)
```

### Published seam (the reusable Fact Builder — what downstream specs consume)

The Fact Builder is **not** a private step of the insights engine: it is a first-class,
exported component, so the deterministic grounding facts can be reused without re-implementation
(BR-1041). The **Conversational Copilot (SPEC-108)** grounds every chat turn through it, and the
Rebalancing Assistant / Health Score (SPEC-105/SPEC-106) reuse it. `*FactBuilder` satisfies
SPEC-108's `FactSource` interface by construction.

```go
// FactBuilder composes the deterministic fact snapshot for a user from the input ports above.
// Exported and stable — the published grounding seam other AI features consume (FR-1041).
type FactBuilder interface {
    BuildFacts(ctx context.Context, userID string) (insight.Facts, error)
}
```

---

## 8. Data Model

**None.** SPEC-104 introduces no tables. The Insighter's in-memory cache (SPEC-005) provides
short-term reuse. Persisted **insight history (FR-022)** is deferred (D4).

---

## 9. Edge Cases

| Scenario | Expected behavior |
| -------- | ----------------- |
| Empty portfolio (no holdings) | Friendly empty state, no LLM call (FR-1047). |
| Profile not set | Build facts without profile-specific fields; still produce portfolio/market insights. |
| LLM unavailable (all categories) | "Insights temporarily unavailable" state (degraded), dashboard unaffected. |
| One category degrades, others succeed | Partial result — return the successful categories. |
| Stale market data (FII without a quote) | Facts note staleness (from the dashboard's `stale_tickers`); insights may mention it. |
| Macro indicator missing (e.g. IFIX gap) | That fact is omitted; market-context insight uses the available indicators. |
| Insighter rejects an output (gate) | That insight is dropped by the gate (SPEC-005); only gated output is surfaced. |

---

## 10. Security Considerations

- **Isolation** — facts are built for the context `user_id`; no cross-user data; identity from
  the session only.
- **AuthN** — `/insights` requires a valid session (not on the public allowlist).
- **No prompt/PII in telemetry** — guaranteed by the Insighter (SPEC-005 BR-505); the engine's
  own spans carry no facts/PII either.
- **Non-advice is a product-safety control** — enforced by the gate, fail-closed (SPEC-005);
  the engine cannot emit ungated AI text.
- **Prompt-injection surface** — holdings/profile values flow into the facts; the gate is the
  backstop, and the Fact Builder passes structured values (not free narrative) where possible.

---

## 11. Observability

- **Traces** — `GET /insights` route span; a `insight.facts` span for fact-building (no PII);
  the Insighter's `insight.generate` spans per category (provider/model/outcome/cache-hit,
  no content).
- **Logs** — `user_id` + `request_id`; never facts or generated text.
- **Metrics** — the Insighter's generation counter by outcome (SPEC-005); optional
  `insights.requests` by outcome (success / partial / unavailable / empty).

---

## 12. Testing Strategy

### Unit Tests

- **Fact Builder** (the heart): table-driven — known dashboard/profile/macro → expected,
  deterministic facts (money centavos / bps); empty portfolio; profile-not-set; missing macro.
- **Engine** with hand-written fakes (a deterministic `insight.Fake` Insighter + fake readers):
  aggregation across categories, the empty state, degradation (Insighter returns
  `ErrInsightsUnavailable` → unavailable state), partial success.
- **Handler**: identity-from-context, empty `200`, degraded state, 401.
- A test asserting the engine emits AI text **only** via the Insighter (no other source).

### Integration Tests (gated)

- Real Postgres + the **`fake` Insighter**: seed holdings + quotes + profile + macro, call
  `GET /insights`, assert every returned insight carries an explanation and the disclaimer is
  present (the gates hold end-to-end).

### Quality gate

`task vet`, `task test:short`, gofmt-clean; integration serialized when a DB is present.

---

## 13. Definition of Done

- [ ] FR-1041…FR-1050 implemented; BR-1041…BR-1047 respected; acceptance criteria met.
- [ ] Fact Builder deterministic; money int64 centavos / bps; the engine emits AI text only via
      the Insighter (gates by construction); identity from context.
- [ ] Hexagonal layering (engine composes read seams + the Insighter port, acyclic, pure core);
      conventions; OpenAPI in lockstep.
- [ ] Unit + gated integration (with the fake Insighter) green; quality gate clean; hexagonal +
      go-correctness reviews pass; suggest `/security-review` (AI-output safety).
- [ ] Closeout: `CHANGELOG`, `README`, SPEC + PLAN → **Done**, indexes, PT-BR lesson.

---

## 14. Decisions (resolved)

| # | Decision | Resolution |
| - | -------- | ---------- |
| D1 | Where the engine lives | **In `internal/insight`** (the existing feature, which owns the Insighter port + gates from SPEC-005) — the engine + Fact Builder are its application layer, reading the dashboard/profile/macro seams. Acyclic, core stays pure. |
| D2 | One fact set, N category calls | **Build one** deterministic fact set and call the Insighter **once per category** (portfolio, allocation, market_context) with a distinct `Task`; the SPEC-005 cache keys by (user, task, facts) so repeats are cheap. |
| D3 | Reuse the dashboard computation | **Yes** — the engine reads `dashboard.Service.GetDashboard` for the allocation/sector/concentration facts (SPEC-103's forward note), rather than recomputing. |
| D4 | Insight history / persistence (FR-022) | **Deferred** to a focused follow-up. The conversational *memory* the user wanted is delivered by the **Conversational Copilot (SPEC-108, FR-025)** — bounded, clearable threads. SPEC-108 D6 confirms FR-022 (per-category *insight* history, this spec's domain) and FR-025 (*conversation* threads) are **distinct memories with no storage overlap**, so insight-history stays a separate, optional concern. SPEC-104 ships the on-demand engine; its `BuildFacts` seam keeps it persistence/reuse-ready (§7). |
| D5 | Degraded HTTP shape | `GET /insights` returns **`200` with `available:false`** + empty items on a full LLM outage (friendlier for the UI than a 503), and a partial `200` when only some categories fail. |
| D6 | Fact Builder as a published seam | **Exported `FactBuilder` / `BuildFacts(ctx, userID)`** (§7) — the reusable grounding source SPEC-108 (`FactSource`) and SPEC-105/106 consume, not a private engine step. |

---

## 15. Open Questions (deferred, not blocking)

> **Forward note — the Fact Builder is the AI grounding seam.** The `BuildFacts(ctx, userID)`
> port published here is the single, deterministic grounding source for the whole AI feature
> set: the Conversational Copilot (SPEC-108) grounds each chat turn through it, and the
> Rebalancing Assistant (SPEC-105) and Health Score (SPEC-106) reuse/extend it. It is also the
> seam the Phase-2 multi-agent CIO + Phase-3 MCP evolve (a tool-call loop would replace the
> pre-built snapshot behind this same port — ADR-0005). Keep it exported and stable.


- Whether allocation insights should compare against an explicit **target allocation** (from a
  future contribution plan / SPEC-107) vs reasoning qualitatively against the risk profile —
  qualitative for MVP; revisit when targets exist.
- Insight **history** (FR-022) shape if/when added — likely reuses the cache's facts hash as a
  dedup key.
- Streaming the insights (SSE) for perceived latency — deferred; the cache + a spinner suffice.
- Per-category caching/refresh controls surfaced to the user — deferred.
