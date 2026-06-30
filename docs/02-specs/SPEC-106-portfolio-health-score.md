# SPEC-106 — Portfolio Health Score

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Portfolio Health Score                                 |
| Feature ID   | SPEC-106                                                |
| Version      | 0.1.0                                                  |
| Status       | Approved                                               |
| Author       | Gabigol                                                |
| Last Updated | 2026-06-29                                             |
| Related PRD  | [PRD](../01-product/PRD.md) §Epic 7, FR-012, FR-013, FR-014 |
| Plan         | PLAN-106 (authored next via /plan-new 106)             |
| Governing    | ADR-0002 (hexagonal); reuses dashboard [SPEC-103](SPEC-103-dashboard.md), profile [SPEC-101](SPEC-101-investor-profile.md), portfolio [SPEC-102](SPEC-102-portfolio-management.md) |

---

## 2. Overview

### Purpose

Compute a **single 0–100 Portfolio Health Score** with a **detailed, per-factor breakdown** —
diversification, concentration, liquidity, goal alignment, and risk exposure — giving the investor
one trustworthy, trackable signal of portfolio health (FR-012).

### Business Value

A single number the investor can watch over time, backed by a transparent breakdown of *why*. It
turns the same computed portfolio facts into an at-a-glance health signal — without a manual audit.

### The defining design — a reproducible computed score + an optional AI narrative (layered)

The PRD success metric is **"same inputs → same score + identical explanation."** The resolution is
a **two-layer** design:

- **Core (deterministic, reproducible).** The **score number** and the **structured per-factor
  breakdown** are **computed**, never LLM-generated — the binding rule *"the LLM never invents
  numbers"* and reproducibility both demand it. Crucially, **market data (macro) is an INPUT** to the
  computation, so the score is **market-aware** yet still reproducible: the same `(portfolio, profile,
  market snapshot)` always yields the same score + identical breakdown. The structured breakdown is the
  **explanation of record** the PRD metric refers to.
- **Narrative (gated LLM, optional).** On top of the computed score, a **gated Insighter narrative**
  (SPEC-005) explains the score + breakdown in rich, contextual prose grounded in the live market —
  the "professor" experience. It is grounded in the computed facts (incl. the score itself), gated
  (FR-013/014), and **never changes the number**. It is best-effort: on an LLM outage it degrades away
  and the score + breakdown still stand.

FR-013 (explainability) holds by construction at both layers (the breakdown always explains; the
narrative is gated). FR-014 (non-advice) holds: the computed breakdown cannot contain an order, and
the narrative passes the order-signature gate.

### Scope

**In scope**

- A **deterministic scoring engine** producing a `0–100` score from five factors, each with a
  sub-score, a weight, and a computed human-readable explanation — **market-aware** (macro is an
  input to the goal-alignment + risk-exposure factors via a documented, modest, tunable rule).
- Reuse of the computed dashboard (SPEC-103, allocation/sector/concentration), the profile
  (SPEC-101, risk/objectives/horizon), the holdings (SPEC-102, liquidity types/counts), and the
  latest macro (SPEC-006, SELIC/CDI/IPCA).
- An **optional gated LLM narrative** (the "professor") explaining the computed score + breakdown
  using the live market, via the SPEC-005 Insighter — grounded, gated, and degradable; it never
  changes the number.
- `GET /health-score` (auth-scoped); the core is reproducible; no persistence.

**Out of scope**

- **Historical tracking** of the score over time (PRD lists it as a future item) — no new tables.
- Any LLM influence on the **score number** or the structured breakdown — those stay computed.
- Insights (SPEC-104), Rebalancing (SPEC-105), Projections (SPEC-107), Chat (SPEC-108).

---

## 3. Functional Requirements

### FR-1061 — Deterministic 0–100 score (reproducible)

Computes an integer score in `[0, 100]` from the five factors. **Same inputs → same score** (the
PRD reproducibility metric); integer arithmetic, documented half-up rounding, never a float.

#### Acceptance Criteria

- [ ] The score is an integer in `[0, 100]`.
- [ ] Identical portfolio + profile inputs always yield the **same score** (determinism test).
- [ ] The weighted combination + rounding is integer-only (no float anywhere).

### FR-1062 — Factor breakdown

The result includes a breakdown of the five factors — **diversification, concentration, liquidity,
goal alignment, risk exposure** — each with its `0–100` sub-score, its weight (bps), and a computed
explanation referencing the facts that drove it.

#### Acceptance Criteria

- [ ] Every factor present carries a sub-score, a weight (bps), and a non-empty explanation.
- [ ] The factor weights sum to `10 000` bps (the score is their weighted mean, reconciles).
- [ ] The explanation cites the relevant computed facts (e.g. "concentração de 45% em Logística").

### FR-1063 — Reproducible breakdown + the gated AI narrative (FR-013/FR-014)

The **structured breakdown is the explanation of record** — computed, not generated, and
**identical for identical inputs** (reproducibility metric). Optionally, a **gated LLM narrative**
explains the computed score + breakdown using the live market; it is grounded in the computed facts
(incl. the score and factor sub-scores), passes the SPEC-005 gates, and **never alters the number**.
The narrative is best-effort: an LLM outage omits it, leaving the score + breakdown intact.

#### Acceptance Criteria

- [ ] The structured breakdown for a given input is byte-identical across runs (reproducibility test).
- [ ] No factor explanation (computed) contains an imperative buy/sell, quantity, price target, or guarantee.
- [ ] When present, the narrative is produced only via the gated `Insighter` (FR-013/014 enforced there)
      and is grounded in the computed score/breakdown/market — it never changes the score.
- [ ] LLM unavailable → the response still returns the score + breakdown, with the narrative omitted.

### FR-1064 — The five factors (deterministic rules)

Each factor is a documented deterministic function of the facts:

- **Diversification** — rewards more holdings spread across more FII sectors / asset classes; a thin
  or single-sector portfolio scores low.
- **Concentration** — inverse of the largest single sector/position share (high concentration → low).
- **Liquidity** — the share of patrimony that is readily liquid (FIIs + daily-liquidity fixed income)
  vs locked at-maturity.
- **Goal alignment** — how close the current allocation is to the **profile-implied, market-aware**
  mix (conservative→FI, aggressive→FII; a documented, modest macro tilt — e.g. higher SELIC shifts
  the "healthy" mix toward post-fixed), reusing the SPEC-105 direction rule extended with the market.
- **Risk exposure** — how well the portfolio's risk posture (concentration + class mix) matches the
  investor's risk tolerance in the current environment.

#### Acceptance Criteria

- [ ] Each factor sub-score is an integer in `[0, 100]`, computed by its documented rule.
- [ ] Each rule is pure (no I/O) and unit-tested with table-driven cases incl. boundaries.
- [ ] The market tilt is **deterministic** (same macro → same tilt) and modest/documented; the score
      stays reproducible given `(portfolio, profile, macro)`.

### FR-1065 — API

`GET /health-score` returns the score, the factor breakdown, and (when available) the AI narrative
+ the non-advice disclaimer. Auth-protected; identity from context.

#### Acceptance Criteria

- [ ] `GET /health-score` → `200` with `{score, factors:[{name, score, weight_bps, explanation}],
      narrative, narrative_available, disclaimer}`.
- [ ] `narrative_available:false` (narrative empty) on an LLM outage; the score + factors are always present.
- [ ] Unauthenticated → `401`; documented in `api/openapi.yaml` (drift test); money/scores are integers.

### FR-1066 — Edge cases (empty portfolio, profile not set)

- **Empty portfolio** → a defined empty state (`score: 0` + an explanation to add holdings), not an error.
- **Profile not set** → the goal-alignment and risk-exposure factors degrade to a documented neutral
  basis (or are omitted with the remaining weights renormalized), explained; the score is still computed.

#### Acceptance Criteria

- [ ] Empty portfolio → `200`, `score: 0`, explanation present (no panic, no 404).
- [ ] Profile-not-set → score still computed; the affected factors explain the missing profile.

### FR-1067 — Observability

`GET /health-score` is traced. **No PII** (no figures, holdings, or profile) on spans/logs (BR-505).

#### Acceptance Criteria

- [ ] A span-content test asserts no fact values / score / holdings on spans.

### FR-1068 — Documentation

Closeout updates `README`, `CHANGELOG`, `api/openapi.yaml`, and a PT-BR lesson.

#### Acceptance Criteria

- [ ] `docs/lessons/SPEC-106-aula.html` produced; OpenAPI in lockstep.

---

## 4. User Flows

### Main Flow

1. The authenticated investor requests `GET /health-score`.
2. The engine reads the dashboard (allocation/sectors/concentration), the profile, and the holdings.
3. It computes each of the five factor sub-scores, the weighted total, and a templated explanation.
4. It returns the score + the per-factor breakdown — identical for identical inputs.

### Alternative Flows

- **Empty portfolio** → `score: 0` with an "add holdings" explanation.
- **Profile not set** → the goal/risk factors note the missing profile; the score is still produced.

---

## 5. Business Rules

### BR-1061

The score and every sub-score are **integers in `[0, 100]`**; factor weights are integer basis
points summing to `10 000`; the weighted combination uses the documented half-up rule
(`internal/platform/money`). **No float** anywhere — the score is reproducible to the unit.

### BR-1062

The **score number and the structured breakdown are computed, not generated** — and are **identical
for identical inputs**, where the inputs include the macro snapshot (the reproducibility metric maps
to this structured explanation of record). The **LLM narrative** is an additive layer that explains
the computed result; it is emitted **only** through the gated `Insighter`, never produces or alters
the score, and may be absent (degraded). FR-013/FR-014 hold at both layers.

### BR-1063

The **scoring core is pure** (no SQL/HTTP/LLM/time): a deterministic function of the structured
inputs `(portfolio, profile, macro)`. Reads + the Insighter call happen at the edge (the service
composes the dashboard/profile/holdings/macro reads, computes the score, then requests the narrative).

### BR-1064

Identity comes from the session context (`auth.UserID(ctx)`); per-user scoping is inherited from the
dashboard/profile/portfolio reads. No client-supplied `user_id` is trusted.

---

## 6. Domain Model

### Value objects / produced (no persistence)

- **`Factor`** — closed enum `diversification | concentration | liquidity | goal_alignment |
  risk_exposure` (typed string constants + `ParseFactor`).
- **`FactorScore`** — `Factor`, `Score` (0–100), `WeightBps`, `Explanation`.
- **`HealthScore`** — `Score` (0–100), `Factors []FactorScore`. The total is the weighted mean of the
  factor sub-scores (half-up); no new tables.

### Inputs (structured, read at the edge)

- The computed **`dashboard.Dashboard`** (allocation, sectors, concentration) — SPEC-103.
- The **`profile.Profile`** (risk/objectives/horizon) — SPEC-101.
- The **holdings** (counts, liquidity types) — SPEC-102.

---

## 7. Ports

### Reused (consumed)

- **`DashboardReader`** — `GetDashboard(ctx, userID)` (SPEC-103).
- **`ProfileReader`** — `GetProfile(ctx, userID)` (SPEC-101).
- **`HoldingsReader`** — `ListHoldings(ctx, userID)` (SPEC-102, for counts + liquidity).
- **`MacroReader`** — `GetLatestMacroIndicator(ctx, ind)` (SPEC-006, for the market-aware factors + the narrative).
- **`insight.Insighter`** — the gated chain (SPEC-005); a new `health_score` task, for the narrative only.

All consumer-defined in the health package; satisfied by the existing services at the edge. No new tables.

---

## 8. Data Model

No new tables or migrations. The score is computed on demand (history is out of scope).

---

## 9. Edge Cases

- **Empty portfolio** → `score: 0`, explanation "add holdings"; no factor division-by-zero.
- **Profile not set** → goal-alignment + risk factors use the neutral basis (D4); explained.
- **Single holding / single sector** → diversification + concentration score low (the intended signal).
- **All at-maturity fixed income** → liquidity factor scores low; explained.
- **Determinism** → the same `(portfolio, profile, macro)` always produce the same score and
  byte-identical structured breakdown.
- **Missing macro** → the market tilt is neutral (the market-aware factors fall back to the
  profile-only basis), explained; the score is still computed.
- **LLM unavailable** → the narrative is omitted (`narrative_available:false`); the score + breakdown stand.

---

## 10. Security Considerations

- **Auth:** the route requires a valid session; identity from context (SPEC-003).
- **Prompt-injection (narrative only):** the narrative passes holdings/score through the LLM, so the
  non-advice gate is the backstop — but the **score is computed**, so injection can never move the
  number; at worst the narrative degrades.
- **Money/score integrity:** integer score / sub-scores / bps; never a float.
- **No PII in telemetry** (BR-505): no figures, holdings, score, profile, or generated narrative on
  spans/logs (the Insighter telemetry already excludes prompt/PII).

---

## 11. Observability

- **Traces:** the `GET /health-score` route span; a `health.compute` span (no content); the
  Insighter's `insight.generate` span for the narrative (SPEC-005, no content).
- **Metrics (optional):** `health_score.requests` by outcome (ok / empty / narrative-degraded).
- **Logs:** structured, at the edges; never the score, holdings, profile, or narrative values.

---

## 12. Testing Strategy

### Unit Tests

- Each of the five factor rules (table-driven, boundaries: empty, single-sector, all-illiquid, etc.).
- The weighted combination + half-up rounding; weights sum to `10 000`; score ∈ `[0, 100]`.
- The **market tilt** — same macro → same tilt; missing macro → neutral; modest bounded effect.
- **Reproducibility** — the same `(portfolio, profile, macro)` produce the same score **and
  byte-identical** structured breakdown.
- The **narrative** — produced only via the gated Insighter (marker test); LLM outage →
  `narrative_available:false`, score + breakdown intact.
- Empty-portfolio and profile-not-set degradation.
- Handler: identity-from-context, `200` shape, `401`.

### Integration Tests (gated by `testing.Short()` + `TEST_DATABASE_URL`)

- Real Postgres: seed holdings + profile + quotes, `GET /health-score`, assert the score is in range,
  the breakdown explains every factor, the result reconciles, and **two calls return an identical
  score + explanation** (reproducibility end to end); per-user isolation.

### Quality gate

- `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
  hexagonal-reviewer + go-correctness-reviewer pass.

---

## 13. Definition of Done

- [ ] FR-1061…FR-1068 implemented; acceptance criteria met; BR-1061…BR-1064 respected.
- [ ] Score deterministic + reproducible (same inputs → same score + identical explanation); every
      factor explained; integer-only; weights reconcile to `10 000`.
- [ ] `GET /health-score` documented in `api/openapi.yaml` (drift test green); identity from context.
- [ ] CHANGELOG + README updated; SPEC-106 + PLAN-106 → Done; indexes + `CLAUDE.md` status updated.
- [ ] PT-BR lesson produced; both review lenses pass; PR opened and `/pr-review` run.

---

## 14. Decisions (resolved)

| # | Decision | Recommendation |
| - | -------- | -------------- |
| D1 | LLM or deterministic? | **Layered (resolved with the user).** The **score number + structured breakdown stay computed** (reproducibility + the "LLM never invents numbers" rule), but the score is **market-aware** because macro is an *input* (so it adjusts with conditions, still reproducible). On top, an **optional gated LLM narrative** explains the result using the live market (the "professor"), grounded + gated + degradable, never touching the number. The reproducibility metric maps to the structured breakdown; the narrative is a labeled additive layer. |
| D2 | Where the engine lives | A new feature package **`internal/health`** with a pure `Compute` + a thin service composing the dashboard/profile/holdings reads (mirrors the SPEC-103 dashboard shape). |
| D3 | Factor weights | Propose **diversification 25% · concentration 25% · liquidity 15% · goal alignment 20% · risk exposure 15%** (bps, Σ = 10 000) — documented, tunable. Confirm the split. |
| D4 | Profile-not-set handling | **Renormalize** — omit goal-alignment + risk-exposure and rescale the remaining weights to `10 000`, with each omitted factor noted; the score stays meaningful. (Alt: a neutral 50/100 basis for the missing factors.) |
| D5 | Reuse structured sources vs `BuildFacts` | **Read the structured dashboard/profile/holdings directly** (cleaner for a numeric computation than parsing the opaque facts map; the facts seam is for LLM grounding, not needed here). |
| D6 | Empty portfolio | **`200` with `score: 0`** + an "add holdings" explanation (consistent with the friendly-state pattern), not a 404. |

---

## 15. Open Questions (deferred, not blocking)

- **Historical tracking** — persisting the score over time to show a trend is a future feature (PRD
  out-of-scope), and the deterministic core makes it a clean follow-up (store score + factor snapshot).
  Because the score is market-aware, a stored series naturally shows both portfolio and market drift.
- **Weight tuning / personalisation** — the factor weights could later flex with the investor profile
  (e.g. liquidity matters more for a short horizon) rather than being fixed.
- **How aggressive the market tilt should be** — the MVP keeps it modest; user testing may justify a
  stronger (or weaker) macro influence on the score, or extending it beyond goal-alignment/risk.
