# SPEC-107 — Projections (Income & Net Worth)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Projections (Passive Income & Net Worth)               |
| Feature ID   | SPEC-107                                                |
| Version      | 0.1.0                                                  |
| Status       | Done                                                   |
| Author       | Gabigol                                                |
| Last Updated | 2026-06-30                                             |
| Related PRD  | [PRD](../01-product/PRD.md) §Epic 8 & 9, FR-016, FR-017, FR-014 |
| Plan         | PLAN-107 (authored next via /plan-new 107)             |
| Governing    | ADR-0002 (hexagonal); reuses dashboard [SPEC-103](SPEC-103-dashboard.md), portfolio [SPEC-102](SPEC-102-portfolio-management.md), market data [SPEC-006](SPEC-006-marketdata-port-and-ingestion-worker.md) |

---

## 2. Overview

### Purpose

Give the investor two forward-looking views over the current portfolio: a **Passive Income
Projection** (monthly/annual income across pessimistic / base / optimistic scenarios, FR-016) and a
**Net-Worth Projection** (wealth over a configurable horizon from current value + reinvested income
+ a configurable monthly contribution, FR-017) — each with its **assumptions shown** and labelled a
non-guaranteed **estimate**.

### Business Value

It turns "what will this produce?" and "where does this get me?" into concrete, chartable numbers —
the planning layer on top of the dashboard, and the data the Conversational Copilot (SPEC-108) reads
for "tenho R$X pra aportar, e daqui a 10 anos?" turns.

### The defining constraint — deterministic & reproducible

FR-016 requires the calculation to be **deterministic and reproducible**, and FR-017 output to be
**chartable**. So — like the Dashboard (SPEC-103) and the Health Score core (SPEC-106) — SPEC-107 is
a **pure computation**: same `(holdings, market, contribution, horizon)` → same projection. **No LLM
is involved** (D1). The scenario assumptions are documented, exposed parameters. FR-014 (non-advice)
holds by construction: a projection is a labelled estimate, never a transaction order, and computed
figures cannot contain an order signature.

### Scope

**In scope**

- A **passive-income projection**: monthly + annual income across three scenarios, from the FII
  dividend income (reusing the dashboard) and the fixed-income rates (holdings), with per-scenario
  assumptions exposed.
- A **net-worth projection**: portfolio value over a configurable horizon (years), compounding
  reinvested income + a configurable **monthly contribution**, across the same three scenarios,
  emitted as time-series points suitable for charting.
- `GET /projections` (auth-scoped), parameterised by `monthly_contribution_centavos` + `horizon_years`.

**Out of scope**

- Any **LLM narrative** (a possible Phase-2 layer, §15; SPEC-108 narrates projections conversationally).
- **Capital-appreciation** modelling of FIIs/equity (income-yield + contribution only in the MVP; §15).
- Persisting projections; tax/inflation adjustment (nominal figures only, an assumption shown).
- Insights (104), Rebalancing (105), Health Score (106), Chat (108).

---

## 3. Functional Requirements

### FR-1071 — Passive Income Projection (FR-016)

Estimates **monthly and annual passive income** over the current holdings across **pessimistic /
base / optimistic** scenarios — from FII dividend income + fixed-income rates — with the assumptions
behind each scenario exposed.

#### Acceptance Criteria

- [ ] Returns, per scenario, a monthly and annual income in `int64` centavos + the scenario assumptions.
- [ ] Base scenario reconciles with the holdings + latest market data (FII dividend income from the
      dashboard, FI income from `invested × annual_rate`); pessimistic/optimistic apply a documented
      yield haircut/uplift.
- [ ] Deterministic: same holdings + market → same figures; integer-only, no float.

### FR-1072 — Net-Worth Projection (FR-017)

Projects portfolio **value over a configurable horizon** (years) from the current value + reinvested
income + a **configurable monthly contribution**, across the same three scenarios, as time-series
points suitable for charting, with assumptions shown.

#### Acceptance Criteria

- [ ] Accepts a `monthly_contribution_centavos` (≥ 0) and a `horizon_years` (bounded, e.g. 1–40).
- [ ] Returns, per scenario, an ordered series of `{year, value_centavos}` points from the current
      value to the horizon; recomputes when the contribution changes.
- [ ] Compounds deterministically (documented monthly compounding, half-up); same inputs → same series.

### FR-1073 — Scenarios & assumptions

Every projection exposes its **three scenarios** and the **assumptions** behind each (yield
adjustment, contribution, horizon, nominal/no-inflation) so the estimate is transparent.

#### Acceptance Criteria

- [ ] `Scenario` is a closed enum `pessimistic | base | optimistic`.
- [ ] Each scenario carries its assumptions as structured integer fields (bps adjustments) + a note.
- [ ] The response is clearly labelled an **estimate, not a guarantee** (non-advice, FR-014).

### FR-1074 — Explainability & Non-Advice, by construction (FR-013/FR-014)

The projection is a labelled estimate with its assumptions shown; there is no LLM, so figures are
computed (never invented) and cannot contain a transaction order.

#### Acceptance Criteria

- [ ] The response carries the estimate/non-advice disclaimer.
- [ ] No figure or assumption text constitutes an imperative buy/sell, quantity, price target, or guarantee.

### FR-1075 — API

`GET /projections?monthly_contribution_centavos=&horizon_years=` returns the income + net-worth
projections. Auth-protected; identity from context; money/rates integers, never float.

#### Acceptance Criteria

- [ ] `GET /projections` → `200` with `{income:[…], net_worth:[…], disclaimer}`; sensible defaults when
      the params are omitted (e.g. contribution `0`, horizon `10`).
- [ ] Invalid params (negative contribution, out-of-range horizon, non-integer) → `400`.
- [ ] Unauthenticated → `401`; documented in `api/openapi.yaml` (drift test).

### FR-1076 — Edge cases & observability

- **Empty portfolio** → zero income; the net-worth projection still grows from the contribution alone.
- The route is traced; **no PII** (no figures/holdings) on spans/logs (BR-505).

#### Acceptance Criteria

- [ ] Empty portfolio → `200`, zero income, net-worth = accumulated contributions; no panic.
- [ ] A span-content test asserts no fact values on spans.

### FR-1077 — Documentation

Closeout updates `README`, `CHANGELOG`, `api/openapi.yaml`, and a PT-BR lesson.

#### Acceptance Criteria

- [ ] `docs/lessons/SPEC-107-aula.html` produced; OpenAPI in lockstep.

---

## 4. User Flows

### Main Flow

1. The authenticated investor requests `GET /projections` with a monthly contribution + horizon.
2. The engine reads the current income (dashboard) + holdings (FI rates) + current value.
3. It computes the three income scenarios and the three net-worth series (monthly compounding).
4. It returns the projections + assumptions + the estimate disclaimer.

### Alternative Flows

- **Empty portfolio** → zero income; net worth = the accumulated contributions over the horizon.
- **Invalid params** → `400` before any computation.

---

## 5. Business Rules

### BR-1071

All money is `int64` centavos and rates integer basis points; the projection compounds with the
documented **half-up** rule (`internal/platform/money`). **No float** anywhere — the projection is
reproducible to the centavo.

### BR-1072

The projection is a **pure, deterministic computation** (no SQL/HTTP/LLM/time in the core): same
`(holdings, market, contribution, horizon)` → same result. Reads happen at the edge (the service
composes the dashboard/holdings reads).

### BR-1073

Scenario assumptions are **documented, exposed parameters** (yield haircut/uplift bps), not hidden —
the estimate is transparent (FR-1073). Figures are **nominal** (no inflation/tax) — stated as an assumption.

### BR-1074

Identity comes from the session context (`auth.UserID(ctx)`); per-user scoping is inherited from the
dashboard/holdings reads. No client-supplied `user_id` is trusted; the only client inputs are the
contribution + horizon.

---

## 6. Domain Model

### Value objects / produced (no persistence)

- **`Scenario`** — closed enum `pessimistic | base | optimistic` (typed constants + `ParseScenario`).
- **`ScenarioIncome`** — `Scenario`, `MonthlyCentavos`, `AnnualCentavos`, `Assumptions`.
- **`ScenarioNetWorth`** — `Scenario`, `Points []NetWorthPoint` (`{Year, ValueCentavos}`), `Assumptions`.
- **`Assumptions`** — structured integer fields (e.g. `IncomeYieldAdjBps`, `MonthlyContributionCentavos`,
  `HorizonYears`) + a human-readable note.
- **`Projections`** — `Income []ScenarioIncome`, `NetWorth []ScenarioNetWorth`, `Disclaimer`. No new tables.

---

## 7. Ports

### Reused (consumed)

- **`DashboardReader`** — `GetDashboard(ctx, userID)` (SPEC-103, current value + FII monthly income).
- **`HoldingsReader`** — `ListHoldings(ctx, userID)` (SPEC-102, fixed-income invested + annual rates).

All consumer-defined in the projection package; satisfied by the existing services at the edge. **No
new ports, no Insighter, no new SQL.**

---

## 8. Data Model

No new tables or migrations. Projections are computed on demand.

---

## 9. Edge Cases

- **Empty portfolio** → zero income; net worth = the future value of the monthly contribution stream.
- **Zero contribution + empty portfolio** → a flat zero net-worth series (no panic, no div-by-zero).
- **Long horizon** (e.g. 40 years / 480 months) → integer compounding stays overflow-safe (big.Int
  where needed); bounded horizon input.
- **Zero current income** (no dividends/rates) → income scenarios are zero; net worth grows from
  contributions only.
- **Determinism** → identical inputs always produce identical figures + series.

---

## 10. Security Considerations

- **Auth:** the route requires a valid session; identity from context (SPEC-003).
- **No LLM surface:** the feature never sends portfolio data to an LLM — no prompt-injection vector.
- **Money integrity:** integer centavos / bps end to end; contribution parsed `≥ 0`; horizon bounded;
  never a float on the wire.
- **No PII in telemetry** (BR-505): no figures, holdings, or series on spans/logs.

---

## 11. Observability

- **Traces:** the `GET /projections` route span; optionally a `projection.compute` span (no content).
- **Metrics (optional):** `projections.requests` count.
- **Logs:** structured, at the edges; never the figures, holdings, or series.

---

## 12. Testing Strategy

### Unit Tests

- Income scenarios (base reconciles to dashboard FII income + FI `invested × rate`; haircut/uplift; empty).
- Net-worth compounding (known contribution + rate + horizon → expected series; monthly half-up;
  empty portfolio = future value of contributions; long-horizon overflow safety).
- Scenario parsing; the weighted/blended yield; **reproducibility** (same inputs → identical output); no float.
- Handler: identity-from-context, param parsing (`400` on bad contribution/horizon), `200` shape, `401`.

### Integration Tests (gated by `testing.Short()` + `TEST_DATABASE_URL`)

- Real Postgres: seed holdings + quotes, `GET /projections`, assert the base income reconciles with
  the seeded data, the net-worth series is monotonic for a positive scenario, **two calls return an
  identical projection** (reproducibility end to end), per-user isolation.

### Quality gate

- `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
  hexagonal-reviewer + go-correctness-reviewer pass.

---

## 13. Definition of Done

- [ ] FR-1071…FR-1077 implemented; acceptance criteria met; BR-1071…BR-1074 respected.
- [ ] Projections deterministic + reproducible; income reconciles with holdings/market; net-worth
      compounds correctly; integer-only; assumptions + disclaimer shown.
- [ ] `GET /projections` documented in `api/openapi.yaml` (drift test green); identity from context.
- [ ] CHANGELOG + README updated; SPEC-107 + PLAN-107 → Done; indexes + `CLAUDE.md` status updated.
- [ ] PT-BR lesson produced; both review lenses pass; PR opened and `/pr-review` run.

---

## 14. Decisions (resolved)

| # | Decision | Recommendation |
| - | -------- | -------------- |
| D1 | LLM or deterministic? | **Deterministic, no LLM.** FR-016 mandates "deterministic and reproducible"; the projection is pure math + shown assumptions. (An optional narrative is a Phase-2 layer / SPEC-108's job — §15.) |
| D2 | One endpoint or two | **One `GET /projections`** returning both income + net-worth — they share inputs and scenarios and are one "planning" view. (Alt: `/projections/income` + `/projections/net-worth` — more surface, no benefit.) |
| D3 | Where the engine lives | A new feature package **`internal/projection`** with a pure `Compute` + a thin service composing the dashboard/holdings reads (mirrors SPEC-106). |
| D4 | Chart granularity | **Compound monthly, emit yearly points** (`{year, value_centavos}`, ≤ 40 points) — enough to chart, cheap payload; contributions/compounding are monthly internally. |
| D5 | Scenario parameters | Propose **base = current blended yield; pessimistic = yield − 200 bps; optimistic = yield + 200 bps** (documented, tunable). Confirm the spread. |
| D6 | Net-worth return basis | **Income-yield + contribution only** for the MVP (reinvested income compounds; no speculative capital appreciation) — conservative + defensible. Appreciation deferred (§15). |
| D7 | Contribution & horizon input | **Query params** on the `GET` (`monthly_contribution_centavos` ≥ 0, `horizon_years` 1–40) with defaults (`0`, `10`); integers, never float. |

---

## 15. Open Questions (deferred, not blocking)

- **An optional LLM narrative** summarising the trajectory ("mantendo R$X/mês, em 10 anos...") could
  layer on later (gated, like SPEC-106) — but the Conversational Copilot (SPEC-108) is the natural
  home for that, reading these computed projections.
- **Capital appreciation** — modelling FII/equity price growth (not just income) would enrich the
  net-worth projection but is more speculative; a documented appreciation assumption could be added.
- **Inflation / real terms** — the MVP is nominal; a real-terms (IPCA-adjusted) view is a future toggle.
- **Saved monthly contribution** — persisting the contribution on the profile (so it prefills here and
  in SPEC-105) is a small future addition.
