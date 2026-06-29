# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Dashboard (portfolio summary, allocation, sector exposure) |
| Feature ID   | SPEC-103 (feature)                                     |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-004, FR-005, Epic 3, §6 Principles, §11 A4/A5, §13 Dependencies |
| Related ADRs | [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) (layering) |
| Version      | 0.1.0                                                  |
| Status       | Done                                                   |
| Plan         | [PLAN-103](../03-plans/PLAN-103-dashboard.md)          |

---

## 2. Overview

### Purpose

Give the investor a single read-only view that **computes**, from their holdings (SPEC-102)
and the latest market data (SPEC-006): the **portfolio summary** (total invested, current
estimated value — i.e. the investor's **full patrimony / net worth** — monthly passive income,
growth — FR-004) and the **allocation breakdown** (by asset class, and FII sector exposure —
FR-005). It is the first feature that *reads across* features and turns stored facts into
derived figures, and it is the **current-state input** the projections feature (SPEC-107)
will grow forward.

### Business Value

"Understand my totals, allocation, and sector exposure at a glance" (Epic 3) is the product's
core read experience, and these are exactly the deterministic facts the AI features
(SPEC-104/105/106) reason over. Getting the money math exact and reproducible here (same
inputs → same figures, PRD §6) is what makes both the dashboard and the future Health Score
trustworthy.

### Scope

**In scope**

- An `internal/dashboard` feature package: the summary/allocation domain + value objects
  (`AssetClass`), the **pure deterministic computation**, the service, and the consumer
  ports it reads through (a holdings reader + a FII quote source).
- One endpoint `GET /dashboard`, behind the deny-by-default auth middleware, per-user.
- Money math (int64 centavos / integer bps, half-up) including a percentage-of-total helper.
- Observability; tests; the working-agreement closeout. **No new tables/migrations** —
  the dashboard is read-only over existing data.

**Out of scope**

- AI insights / health score / rebalancing (SPEC-104/105/106 — they reuse these facts).
- Future-value projections over a horizon (SPEC-107); this computes *current* value only.
- Stocks/ETFs as managed asset classes (allocation buckets appear but are always 0 in MVP).
- Any write/mutation, any new persistence, any AI output (FR-013/FR-014 do not apply).

---

## 3. Functional Requirements

### FR-1031 — Portfolio Summary (FR-004)

#### Acceptance Criteria

- [ ] Computes **total invested** (cost basis: Σ FII `quantity × average_price` + Σ FI
      `invested_amount`), **current estimated value**, **monthly passive income**, and
      **growth** (absolute centavos + relative bps), all as `int64` centavos / integer bps.
- [ ] **Current value:** per FII = `quantity × current market price` (latest quote); per FI =
      `invested_amount` plus a simple-interest accrual to today (PRD A4 approximation, D2).
- [ ] **Monthly passive income:** Σ over FIIs of `last_dividend × quantity` (FIIs distribute
      monthly); a FII with no quote contributes 0.

### FR-1032 — Allocation by Asset Class (FR-005)

#### Acceptance Criteria

- [ ] Returns the current-value share of each asset class — `fii`, `fixed_income`, `stocks`,
      `etfs` — as a value (centavos) and a percentage (**bps of total**). Stocks/ETFs are 0 in
      MVP (no managed holdings) but present for forward-compatibility.
- [ ] Shares are computed half-up and are reconciled (Σ values = total current value).

### FR-1033 — FII Sector Exposure (FR-005)

#### Acceptance Criteria

- [ ] Groups FII current value by **sector** (Logistics, Offices, Shopping, Hybrid, Paper,
      Other — from the quote) and returns each as value + **bps of the FII total**.
- [ ] A held FII with no quote (sector unknown) is grouped under `other` and flagged stale.

### FR-1034 — Deterministic Money Computation

#### Acceptance Criteria

- [ ] All sums/products are `int64` centavos; all percentages integer **bps**; every division
      rounds **half-up** via `internal/platform/money`. The same holdings + quotes always
      produce identical figures (PRD §6) — no `float64` anywhere, including the wire.

### FR-1035 — Reads via Consumer Ports (per-user)

#### Acceptance Criteria

- [ ] The service reads holdings through a consumer-defined holdings reader (satisfied by
      `portfolio.Reader`) and quotes through a FII quote source (satisfied by the marketdata
      quote repository); the domain core stays pure.
- [ ] `user_id` comes from the authenticated context (`auth.UserID(ctx)`), never request input.

### FR-1036 — Graceful Degradation on Stale/Missing Data

#### Acceptance Criteria

- [ ] A FII with no/failed quote falls back to **cost basis** for its current value (so the
      total still reconciles) and its ticker is reported in a `stale_tickers` list.
- [ ] An empty portfolio returns a zeroed summary and empty allocation/sectors with `200`.

### FR-1037 — API Contract & Money on the Wire

#### Acceptance Criteria

- [ ] `GET /dashboard` returns the summary + allocation + sectors; money as integer
      `*_centavos`, percentages as integer `*_bps` — never a float.
- [ ] Requires authentication (not on the public allowlist); errors use the `{"error":"..."}`
      envelope.

### FR-1038 — Observability

#### Acceptance Criteria

- [ ] The endpoint inherits the `otelhttp` route span (`GET /dashboard`); read calls appear as
      child query spans; no PII beyond `user_id`, no money values on spans.

### FR-1039 — Documentation

#### Acceptance Criteria

- [ ] `README` (endpoint) + `CHANGELOG` updated; the PT-BR lesson
      `docs/lessons/SPEC-103-aula.html` produced on close.

---

## 4. User Flows

### Main Flow

1. The authenticated user `GET /dashboard`.
2. The service reads the caller's holdings and the latest quote for each held FII.
3. The pure computation derives the summary, allocation, and sector exposure (deterministic).
4. The response returns the figures (money as integer centavos, shares as bps).

### Alternative Flow — stale market data

1. A held FII has no stored quote.
2. Its current value falls back to cost basis; the total still reconciles; the ticker appears
   in `stale_tickers`.

---

## 5. Business Rules

- **BR-1031 — Facts are computed, not generated.** Every figure is a deterministic function
  of the holdings + quotes; the same inputs always yield the same output (PRD §6). This is the
  binding "facts-computed-not-generated" constraint applied to the read model.
- **BR-1032 — Money is never `float64`.** Sums/products are `int64` centavos, percentages
  integer **bps**; every ratio rounds **half-up** via `internal/platform/money`. The ban
  extends to the JSON wire.
- **BR-1033 — Identity from context.** `user_id` from the session; per-user reads via the
  consumer ports (no client-supplied identity).
- **BR-1034 — Reconciliation.** The summary's current value equals the sum of the per-holding
  current values, and each allocation/sector breakdown sums (by value) to its parent total —
  the figures are auditable against the underlying holdings (Epic 3 acceptance).
- **BR-1035 — Cost basis vs market value.** Invested is the holdings' cost basis (SPEC-102);
  current value uses the latest market price (FII) or a simple-interest accrual (FI, PRD A4 —
  *not* mark-to-market); a missing quote falls back to cost basis (BR-1036/FR-1036).
- **BR-1036 — No AI output, but this is the substrate.** No LLM here (FR-013/FR-014 N/A); the
  deterministic facts computed here are exactly what the Fact Builder (SPEC-104) reuses and
  extends for the Insighter.
- **BR-1037 — Read-only.** No new tables or migrations; the dashboard composes existing read
  seams (`portfolio.Reader` + the marketdata quote repo), an acyclic feature dependency.
- **BR-1038 — Conventions.** Errors `%w` + sentinels; `Clock` over `time.Now()` (the FI
  accrual + growth use it); `ctx` first; DTOs separate from domain; doc comments cite SPEC/BR;
  no package-name stutter.

---

## 6. Domain Model

> Read model only — no persisted entities.

### Value object: AssetClass

Closed enum: `fii` | `fixed_income` | `stocks` | `etfs`.

### Computed structures

| Type | Fields |
| ---- | ------ |
| `Summary` | `TotalInvestedCentavos int64`, `CurrentValueCentavos int64` (**the full patrimony / net worth — the sum of every holding's current value**), `MonthlyIncomeCentavos int64`, `GrowthCentavos int64`, `GrowthBps int` |
| `ClassSlice` | `Class AssetClass`, `ValueCentavos int64`, `ShareBps int` |
| `SectorSlice` | `Sector marketdata.Sector`, `ValueCentavos int64`, `ShareBps int` |
| `Dashboard` | `Summary`, `Allocation []ClassSlice`, `FIISectors []SectorSlice`, `StaleTickers []string` |

Sector reuses `marketdata.Sector`; `Ticker` reuses `marketdata.Ticker` (consistent with
SPEC-102 D1).

---

## 7. Ports (consumer-defined)

```go
// dashboard reads holdings and quotes through small consumer interfaces it defines (the
// "accept interfaces" convention); portfolio.Service and the marketdata quote repo satisfy them.
type HoldingsReader interface {
    ListHoldings(ctx context.Context, userID string) (portfolio.Holdings, error)
}

type QuoteSource interface {
    GetFIIQuoteByTicker(ctx context.Context, t marketdata.Ticker) (marketdata.FIIQuote, error) // marketdata.ErrFIIQuoteNotFound when absent
}
```

> No repository/Reader of its own to expose downstream — the dashboard *is* a consumer. The
> pure computation is a separate, side-effect-free function the service calls.

---

## 8. Data Model

**None.** The dashboard introduces no tables and no migration; it reads `fii_holdings`,
`fixed_income_holdings` (SPEC-102), and `fii_quotes` (SPEC-006) through the ports above.

---

## 9. Edge Cases

| Scenario | Expected behavior |
| -------- | ----------------- |
| Empty portfolio | Zeroed summary, empty allocation/sectors, `200`. |
| FII held but no stored quote | Current value = cost basis; ticker in `stale_tickers`; sector → `other`. |
| Total current value is 0 | All shares are 0 bps (no divide-by-zero). |
| Invested is 0 but value > 0 (e.g. all cost basis 0) | `GrowthBps` guarded to 0 (no divide-by-zero). |
| FI with daily liquidity / no maturity | Accrues from `created_at` to today like any FI (A4). |
| Quote present but `last_dividend` 0 | Contributes 0 monthly income (valid). |
| Unknown/`other` sector quotes | Aggregated under the `other` sector slice. |

---

## 10. Security Considerations

- **Isolation** — all reads scoped to the context `user_id`; a user only ever sees figures
  derived from their own holdings.
- **AuthN** — `/dashboard` requires a valid session (absent from the public allowlist).
- **Determinism as integrity** — integer centavos end-to-end means no rounding/precision drift
  can silently corrupt a displayed balance.
- **No new secrets, no AI output, no writes.**

---

## 11. Observability

- **Traces** — `otelhttp` route span `GET /dashboard`; `otelsql` child spans for the holdings
  + quote reads (statement only, no argument values).
- **Logs** — `user_id` + `request_id`; never holding/figure values.
- **Metrics** — optional `dashboard.requests` counter by outcome; no PII.

---

## 12. Testing Strategy

### Unit Tests

- **The pure computation** (table-driven, the heart of this spec): known holdings + quotes →
  expected summary/allocation/sectors, asserting **reconciliation** (Σ slices = totals) and
  **determinism**; stale-quote fallback; empty portfolio; divide-by-zero guards; FI accrual
  with a fake `Clock`.
- Service with hand-written fakes (holdings reader + quote source), incl. a missing quote.
- Handler: money-as-integer-centavos round-trip, empty portfolio `200`, identity-from-context.

### Integration Tests (gated)

- Real Postgres (`TEST_DATABASE_URL`, `-p 1`): seed a user's holdings + `fii_quotes`, call the
  service, assert the computed figures reconcile end-to-end across SPEC-102 + SPEC-006 data.

### Quality gate

`task vet`, `task test:short`, gofmt-clean; integration serialized when a DB is present.

---

## 13. Definition of Done

- [ ] FR-1031…FR-1039 implemented; BR-1031…BR-1038 respected; acceptance criteria met.
- [ ] Hexagonal layering (pure computation core; consumer ports; HTTP in transport); money
      int64 centavos / bps end-to-end incl. the wire; deterministic + reconciling; identity
      from context; conventions honored.
- [ ] Computation proven deterministic + reconciling; stale-data fallback proven; gated
      integration green against real Postgres.
- [ ] Unit + integration green; quality gate clean; hexagonal + go-correctness reviews pass.
- [ ] Closeout: `CHANGELOG`, `README` (endpoint), SPEC + PLAN → **Done**, indexes, PT-BR lesson.

---

## 14. Decisions (resolved)

| # | Decision | Recommendation |
| - | -------- | -------------- |
| D1 | Dashboard as a new `internal/dashboard` feature package composing `portfolio.Reader` + a marketdata quote port | **Yes** — the Reader/quote ports exist precisely for downstream consumers; the dependency is acyclic (a higher-level read model over portfolio + marketdata). The pure computation keeps the core decoupled. |
| D2 | FI current value | **Simple-interest accrual to today**: `invested + invested × annual_rate_bps × elapsed_days / (10000 × 365)`, half-up — the PRD A4 approximation. (Alternative: cost-basis only, deferring accrual to SPEC-107.) |
| D3 | Monthly passive income | **Σ FII `last_dividend × quantity`** (FIIs distribute monthly) — direct and "monthly". FI interest is not a monthly distribution and is excluded from this figure. |
| D4 | Missing FII quote | **Fall back to cost basis + list the ticker in `stale_tickers`** (total still reconciles), over excluding the holding or erroring. |
| D5 | Allocation classes | **All four** (`fii`, `fixed_income`, `stocks`, `etfs`) with Stocks/ETFs = 0 in MVP — matches the FR-005 enumeration and is forward-compatible. |

---

## 15. Open Questions (deferred, not blocking)

- Whether to surface a portfolio-level **freshness timestamp** (oldest quote `fetched_at`)
  alongside `stale_tickers` — easy add once the dashboard UI needs it.
- FI accrual model (simple vs compound) — simple for MVP (A4); revisit with projections
  (SPEC-107), which may share the accrual helper.
- Reusing this computation as the SPEC-104 **Fact Builder** core (likely) vs duplicating —
  resolve when SPEC-104 is specced; the pure functions are designed to be reusable.
- A combined endpoint or caching if the dashboard becomes read-heavy (premature now).

### Forward note — patrimony projection (SPEC-107, out of scope here)

The dashboard's current patrimony (`current_value_centavos`) is the **starting point** for the
net-worth/passive-income **projection** feature (FR-016/FR-017, SPEC-107), which will grow it
forward over a horizon under **optimist / moderate / pessimist** scenarios. SPEC-107 will read
the same holdings + market-data seams this spec uses, plus two new persisted inputs the owner
requested:

- a **contribution plan** — a monthly contribution amount (centavos) **plus a per-asset-class
  split** (FII / Fixed Income / Stocks / ETFs, in bps — the same four classes this dashboard's
  allocation reports; "CDB" is a Fixed Income *instrument*, not a separate class), and
- a **scenario selector** (optimist / moderate / pessimist).

Recommended placement (to settle in SPEC-107): a small `ContributionPlan` owned by SPEC-107,
**separate from the investor profile** (SPEC-101 captures *who you are*; a contribution plan is
*how you invest going forward*). None of this is built in SPEC-103 — captured here so the
dashboard's classes and net-worth figure are designed to feed it cleanly. The per-class
contribution split is a small refinement to fold into the PRD's FR-016/FR-017 when SPEC-107 is
specced.
