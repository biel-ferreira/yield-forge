# SPEC-109 — Fixed-Income Rate Indexers (% do CDI / IPCA+)

## 1. Document Information

| Field        | Value                                                    |
| ------------ | --------------------------------------------------------- |
| Feature Name | Fixed-Income Rate Indexers                                |
| Feature ID   | SPEC-109                                                  |
| Version      | 0.1.0                                                     |
| Status       | Approved                                                  |
| Author       | Gabigol                                                  |
| Last Updated | 2026-07-02                                                |
| Related PRD  | [Epic 1 / FR-002](../01-product/PRD.md) — refines the fixed-income holding fields to match how Brazilian retail investors actually quote a rate |
| Governing    | Extends [SPEC-102](SPEC-102-portfolio-management.md) (fixed-income domain), reuses [SPEC-006](SPEC-006-marketdata-port-and-ingestion-worker.md) (already-ingested SELIC/CDI/IPCA); consumed by [SPEC-103](SPEC-103-dashboard.md) and [SPEC-107](SPEC-107-projections.md) (FI accrual math); precedes [SPEC-211](SPEC-211-portfolio-management-screens.md) (frontend) |

---

## 2. Overview

### Purpose

Today a fixed-income holding stores one flat `annual_rate_bps`. In practice, most Brazilian
retail fixed income (CDBs, LCs, LCAs) is quoted as **"% do CDI"** (pós-fixado), and a large share
of the rest as **"IPCA + X%"** (híbrido, e.g. Tesouro IPCA+) — a flat rate (**prefixado**) is
actually the minority case. This spec adds a **rate indexer** to the fixed-income holding so the
investor can enter the rate the way their bank statement shows it, and lets the backend **resolve
the effective current annual rate** using the SELIC/CDI/IPCA data SPEC-006 already ingests — so the
number stays correct as the reference rate moves, instead of going stale the moment it's entered.

### Business Value

This closes a real gap surfaced while drafting SPEC-211 (Portfolio Management screens): asking a
user for "the annual rate" when they only know "120% do CDI" is a bad question, and silently
converting it once at entry would produce a number that quietly drifts wrong every time the
Central Bank moves SELIC. Resolving it live keeps the reproducibility promise this project holds
elsewhere (Dashboard, Health Score) — the effective rate is a **derived fact**, not typed once and
forgotten.

### Success Criteria

- A fixed-income holding can be **prefixado** (flat, today's behavior, unchanged), **% do CDI**, or
  **IPCA + X%**.
- The **effective current annual rate** is resolved at read-time from the persisted indexer +
  the latest ingested SELIC/CDI/IPCA (never stored as a stale snapshot) and is what the Dashboard
  (SPEC-103) and Projections (SPEC-107) use for accrual/income math.
- Existing holdings are **unaffected** — they default to `prefixado`, byte-for-byte the same
  behavior as before this spec.
- A new read-only endpoint exposes the latest SELIC/CDI/IPCA so a client can show the reference
  rate without duplicating BCB calls.

---

## 3. Functional Requirements

### FR-1091 — Persist a rate indexer on the fixed-income holding

Add a closed-enum `indexer_type` (`prefixado` | `cdi_percentual` | `ipca_spread`) to
`FixedIncomeHolding`, defaulting existing rows to `prefixado`. The existing `annual_rate_bps` field
is **reinterpreted per indexer**: for `prefixado` it is the flat annual rate (unchanged); for
`cdi_percentual` it is the percentage of CDI, in bps-of-percent (e.g. `12000` = 120.00%); for
`ipca_spread` it is the spread over IPCA, in bps (e.g. `580` = +5.80%).

#### Acceptance Criteria

- [ ] `indexer_type` defaults to `prefixado`; migration is additive and backward-compatible (no
      existing row's stored value or meaning changes).
- [ ] Value validates in its constructor (parse-don't-validate, mirrors `LiquidityType`); an
      unknown value is a `400`, never silently coerced.
- [ ] Money/rate stays integer end to end — no `float64` anywhere in this feature.

### FR-1092 — Resolve the effective annual rate at read-time

A new computation resolves the **effective annual rate** from the holding's indexer + the latest
`MacroIndicator` (SPEC-006): `prefixado` passes `annual_rate_bps` through unchanged;
`cdi_percentual` computes `annual_rate_bps × latest(CDI)_bps ÷ 10000`; `ipca_spread` computes
`annual_rate_bps + latest(IPCA)_bps`. All rounding half-up, overflow-safe (`big.Int`), mirroring
the existing `money` helper conventions. Exposed on `FixedIncomeResponse` as a new, **computed,
never-persisted** `effective_annual_rate_bps`.

#### Acceptance Criteria

- [ ] `prefixado` → `effective_annual_rate_bps == annual_rate_bps`, always.
- [ ] `cdi_percentual` / `ipca_spread` → correctly resolved against the latest stored macro value
      (table-tested against known fixtures).
- [ ] The resolution is a pure function of `(indexer, rate_value, macro_snapshot)` — same inputs,
      same output, every time (reproducibility, mirrors BR-1064/BR-1092 elsewhere in the project).

### FR-1093 — Dashboard FI accrual uses the effective rate

The SPEC-103 Dashboard's FI current-value accrual (`invested + invested × rate × elapsed_days ÷
(10000 × 365)`) uses the **effective** annual rate (FR-1092) instead of assuming the stored value
is already a flat rate.

#### Acceptance Criteria

- [ ] A `prefixado` holding's computed current value is **unchanged** from today (regression-safe).
- [ ] A `cdi_percentual`/`ipca_spread` holding's current value reflects the resolved effective rate.

### FR-1094 — Projections FI income uses the effective rate

The SPEC-107 Projections' fixed-income income calculation uses the resolved effective rate for
every scenario (pessimistic/base/optimistic), same resolution as FR-1092/1093.

#### Acceptance Criteria

- [ ] `prefixado` projections are unchanged (regression-safe).
- [ ] Indexed holdings project using the effective rate resolved at the time of the request.

### FR-1095 — Expose the latest macro indicators (read-only)

A new `GET /market/indicators` endpoint returns the latest SELIC, CDI, and IPCA (reusing the
existing SPEC-006 `MacroRepository` — no new ingestion), so a client can show "CDI atual: 10,50%
a.a." next to an indexer picker without calling BCB directly.

#### Acceptance Criteria

- [ ] Returns the latest value + reference date for SELIC/CDI/IPCA, rates as integer bps.
- [ ] Behind the standard deny-by-default auth gate (consistent posture; the data itself carries
      no PII and is not user-scoped).
- [ ] Documented in `api/openapi.yaml`; the drift test passes.

---

## 4. User Flows

### Main Flow (create a % do CDI holding)

1. A `cdi_percentual` holding is created with `annual_rate_bps = 12000` (120% do CDI).
2. On the next Dashboard read, the effective rate resolves against the latest ingested CDI
   (e.g. `1050` bps / 10.50% a.a.) → effective `1260` bps / 12.60% a.a. — used for accrual.
3. Next month, CDI moves (BCB/SELIC decision) → the **same holding, unchanged**, now resolves a
   different effective rate automatically, with no user action.

### Alternative Flow (prefixado, unaffected)

1. An existing (or new) `prefixado` holding's effective rate always equals its stored
   `annual_rate_bps` — identical to the pre-SPEC-109 behavior.

---

## 5. Business Rules

### BR-1091 — Money & rates stay integer, including the resolution math
The stored value and the resolved effective rate are both integer basis points; the resolution
arithmetic (multiply/divide/add) never introduces a float (BR-1022 extended).

### BR-1092 — The effective rate is a derived fact, never stored
Mirrors the project's "facts are computed, not generated" principle (and the Health Score's
reproducibility bar): the same `(indexer, rate_value, macro_snapshot)` always yields the same
effective rate; nothing about it is cached as a point-in-time snapshot on the holding itself.

### BR-1093 — Backward compatibility is non-negotiable
Every holding written before this spec ships defaults to `prefixado` and produces **byte-for-byte
identical** Dashboard/Projection output to before this spec (regression-tested).

### BR-1094 — Graceful degradation on missing/stale macro data
If the relevant indicator (CDI/IPCA) has no ingested value yet, resolution falls back to the same
**last-known-good** posture SPEC-006 already uses elsewhere (never crash, never silently return
zero) — the specific fallback shape (return the raw un-resolved value vs. an explicit
"unavailable" flag) is a PLAN-109 decision, resolved with a fake `Clock`/fixture in tests.

### BR-1095 — Identity/ownership and market-data scoping unchanged
Fixed-income holdings keep the existing SPEC-102 per-user ownership rules unchanged. Market
indicators remain **global reference data** — no `user_id` (mirrors SPEC-006's BR-603).

---

## 6. Domain Model

### Entity: FixedIncomeHolding (extended)

| Field              | Type          | Description                                             |
| ------------------ | ------------- | --------------------------------------------------------- |
| indexer_type        | Indexer       | `prefixado` \| `cdi_percentual` \| `ipca_spread` (new)     |
| annual_rate_bps     | int64 (bps)   | Meaning depends on `indexer_type` (existing column, reinterpreted) |

### Value Object: Indexer (new, closed enum)
`prefixado` | `cdi_percentual` | `ipca_spread` — parse-don't-validate, mirrors `LiquidityType` (SPEC-102 D3).

### Computed (never persisted): effective_annual_rate_bps
A pure function of `(indexer_type, annual_rate_bps, latest MacroIndicator)` — resolved at read time
by the Dashboard, Projections, and the holdings read path (FR-1092/1093/1094).

---

## 7. API Contract

### `FixedIncomeRequest` / `FixedIncomeResponse` (extended)
Adds `indexer_type` (enum, default `prefixado`) to both. `FixedIncomeResponse` additionally
exposes the computed `effective_annual_rate_bps` (read-only, never accepted on the request).

### `GET /market/indicators` (new)
```json
[
  { "indicator": "selic", "value_bps": 1075, "reference_date": "2026-07-01" },
  { "indicator": "cdi",   "value_bps": 1050, "reference_date": "2026-07-01" },
  { "indicator": "ipca",  "value_bps": 450,  "reference_date": "2026-06-01" }
]
```
Auth-protected (deny-by-default, consistent posture); no `user_id`.

---

## 8. Data Model

### Migration (new, additive)
`fixed_income_holdings` gains `indexer_type` (`text`/enum, `NOT NULL DEFAULT 'prefixado'`) — no
backfill needed beyond the default; existing rows are correct by construction. No new table for
`GET /market/indicators` — it reads the existing `macro_indicators` table (SPEC-006, migration
`0003_market_data`).

---

## 9. Edge Cases

### Existing holdings pre-dating this spec
Default to `prefixado`; Dashboard/Projections output must be byte-for-byte unchanged
(regression-tested against fixtures captured before this spec).

### Macro indicator missing/stale (fresh environment, ingestion not yet run)
Resolution degrades gracefully (BR-1094) rather than erroring the whole Dashboard/Projections read.

### A holding is created with an indexer the day before its first ingestion run
The effective rate resolves against whatever the latest stored value is at read time — no special
casing needed; this is the same "read whatever's current" posture as everywhere else market data
is consumed.

---

## 10. Security Requirements

### Authentication / Authorization
Holdings keep existing per-user ownership (SPEC-102, unchanged). `GET /market/indicators` sits
behind the same deny-by-default auth gate as every other authenticated route; it is global
reference data, not user-scoped.

### Data Protection
No new secrets or PII; SELIC/CDI/IPCA are public economic indicators.

---

## 11. Observability

Route-named `otelhttp` spans extend automatically to `GET /market/indicators`. No new metrics
required; the resolution function is pure/cheap enough not to warrant its own span.

---

## 12. Testing Strategy

### Unit Tests
- `Indexer` value object (parse-don't-validate, unknown value rejected).
- Effective-rate resolution: `prefixado` passthrough, `cdi_percentual` math, `ipca_spread` math,
  half-up rounding, overflow safety (table-driven fixtures).
- Degradation path when the relevant indicator is missing.

### Integration Tests
- Migration up/down round-trip; existing rows default to `prefixado` correctly.
- Real-Postgres: create a `cdi_percentual` holding, seed a known CDI fixture, assert the resolved
  `effective_annual_rate_bps` on `GET /holdings/fixed-income` and the Dashboard's computed current
  value both reflect it.
- Regression: `prefixado` holdings' Dashboard/Projections output is unchanged before/after this spec.

### E2E / Contract
- `GET /market/indicators` documented and drift-test green.

---

## 13. Definition of Done

- [ ] FR-1091…FR-1095 implemented; BR-1091…BR-1095 respected.
- [ ] Migration up/down proven against real Postgres; backward compatibility regression-tested.
- [ ] `api/openapi.yaml` updated (extended schemas + new endpoint); drift test green.
- [ ] Dashboard (SPEC-103) and Projections (SPEC-107) consume the resolved effective rate.
- [ ] `task vet` + `task test:short` clean; integration tests green against real Postgres.
- [ ] CHANGELOG updated; SPEC-109 + PLAN-109 flipped to Done; indexes updated.
- [ ] PT-BR lesson `docs/lessons/SPEC-109-aula.html` via **lesson-writer** (backend track).

---

## 14. Open Questions

1. **PRD amendment?** Epic 1's acceptance criteria currently say just "annual_rate" for a fixed-
   income holding. This spec refines that to match real-world usage rather than expanding scope,
   but you may want a small PRD note added alongside this spec — your call, not decided here.
2. **Exact degradation shape for missing macro data** (BR-1094) — return the raw un-resolved value,
   or an explicit `"effective_annual_rate_bps": null` / `"rate_status": "stale"`? Left to PLAN-109.
3. **SPEC-211 impact** — once this lands, SPEC-211's FR-2116 (Add a fixed-income holding) should be
   revised to offer the three indexer modes instead of a single flat-rate field, and its FR-2119
   (money input parsing) stays as-is (still needed for the raw amount fields). SPEC-211 is left in
   Draft, unblocked to resume once SPEC-109 is approved.
