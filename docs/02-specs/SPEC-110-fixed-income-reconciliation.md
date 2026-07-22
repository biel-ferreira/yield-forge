# SPEC-110 — Fixed-Income Reconciliation & Portfolio Update Indicator

## 1. Document Information

| Field        | Value                                                    |
| ------------ | --------------------------------------------------------- |
| Feature Name | Fixed-Income Monthly Reconciliation & Portfolio Staleness Indicator |
| Feature ID   | SPEC-110                                                  |
| Version      | 0.1.0                                                     |
| Status       | Approved                                                  |
| Author       | Gabigol                                                  |
| Last Updated | 2026-07-15                                                |
| Related PRD  | [Epic 1 / FR-002](../01-product/PRD.md), [§11 A4](../01-product/PRD.md) (FI current value is an accrual approximation, not mark-to-market) |
| Governing    | Extends [SPEC-102](SPEC-102-portfolio-management.md) (fixed-income domain) and [SPEC-109](SPEC-109-fixed-income-rate-indexers.md) (effective-rate resolution); consumed by [SPEC-103](SPEC-103-dashboard.md)'s dashboard; frontend consequences land in [SPEC-211](SPEC-211-portfolio-management-screens.md) and [SPEC-212](SPEC-212-dashboard-screen.md) (see §15 Forward Note) |

---

## 2. Overview

### Purpose

Today a fixed-income holding's "current value" is a pure formula
(`invested_amount + simple_interest(rate, days since created_at)`, PRD A4) with no way for the
user to confirm it against what their bank statement actually shows, and no way to tell **how
much of a balance increase is a new contribution (aporte) versus interest earned
(valorização)** — editing `invested_amount_centavos` today just overwrites it, conflating the
two. A real bug follows from this: `UpdateFixedIncomeHolding` never resets the accrual clock
(`created_at`), so adding new money to a holding causes the formula to retroactively treat that
money as if it had been earning interest since the holding's original creation date.

This spec adds a **reconciliation** action — a monthly checkpoint where the user confirms (or
adjusts) the interest the system estimates has accrued, and separately reports any new
contribution (which may be zero) — and a **portfolio-wide "did you update this month?"
indicator**, the deliberate stand-in for real bank-statement ground truth until Open Finance
(out of MVP scope, PRD §17) exists.

### Business Value

Without Open Finance, the platform's fixed-income figures are estimates by construction (PRD
A4). Reconciliation makes that estimate self-correcting month over month instead of silently
drifting, and — the sharper problem — makes "how much did I actually earn" answerable at all: a
bare running balance cannot distinguish money the user put in from money the holding earned, but
tagging every balance change at the moment it happens can. The staleness indicator closes the
loop by prompting the update instead of letting the dashboard quietly go stale forever.

### Success Criteria

- A fixed-income holding tracks **money contributed** (`total_contributed_centavos`) separately
  from its **current balance** (`invested_amount_centavos`, reinterpreted — see FR-1101); the two
  only diverge by confirmed/estimated interest, never by ambiguity.
- A dedicated **reconcile** action lets the user confirm/adjust the estimated interest since the
  last reconciliation and report a new contribution (zero allowed), and resets the accrual clock
  correctly — fixing the retroactive-interest bug.
- The existing `PUT` edit path keeps working for metadata/typo corrections, and no longer
  produces incorrect interest when it touches the invested amount.
- The Dashboard exposes **per-asset-class growth** (FII vs. Fixed Income) instead of one blended
  total-patrimony figure, and a **reconciliation-due** signal per stale fixed-income holding.
- The Dashboard exposes **per-holding current value** (each FII ticker, each fixed-income
  holding) in R$, not just each asset class's percentage share — today's `GET /dashboard` has no
  way to show "how much of my patrimony is in HGLG11" or "how much is in this specific CDB"
  beyond the class/sector aggregate.
- Existing holdings are **unaffected**: byte-for-byte identical Dashboard output until their
  first reconciliation (mirrors SPEC-109 BR-1093).

---

## 3. Functional Requirements

### FR-1101 — Separate lifetime contribution from current balance

Add two fields to `FixedIncomeHolding`: `TotalContributedCentavos int64` (the lifetime sum of
money the user has actually put in — the true cost basis for growth) and `LastReconciledAt
time.Time` (the accrual clock — replaces `CreatedAt` for interest-accrual purposes).
`InvestedAmountCentavos` is **reinterpreted** (mirrors how SPEC-109 reinterpreted
`annual_rate_bps`): it becomes the current principal/balance basis used for accrual, which grows
via both contributions **and** confirmed interest — it is no longer, by itself, the cost basis.

#### Acceptance Criteria

- [ ] Migration is additive; every existing row backfills `total_contributed_centavos =
      invested_amount_centavos` and `last_reconciled_at = created_at` — correct by construction,
      no ambiguity for pre-spec holdings (mirrors SPEC-109 BR-1093).
- [ ] `CreateFixedIncomeHolding` sets `TotalContributedCentavos = InvestedAmountCentavos` and
      `LastReconciledAt = CreatedAt` for a new holding (the opening balance is its first
      contribution).
- [ ] Money stays integer end to end — no `float64` anywhere in this feature (BR-1022).

### FR-1102 — Fix the accrual-clock reset bug on plain edits

`UpdateFixedIncomeHolding` (existing `PUT`) resets `LastReconciledAt` to now whenever the request
changes `InvestedAmountCentavos`, so a correction never back-dates new principal's interest to the
holding's original creation date. A plain edit does **not** change `TotalContributedCentavos` — it
is a correction, not a contribution.

#### Acceptance Criteria

- [ ] Editing only `name`/`institution`/`annual_rate_bps`/`indexer_type`/`maturity_date`/
      `liquidity_type` leaves `LastReconciledAt` unchanged.
- [ ] Editing `invested_amount_centavos` (any direction) resets `LastReconciledAt` to now.
- [ ] Regression test reproduces today's bug (edit → retroactive interest) and proves it fixed.

### FR-1103 — Monthly reconciliation action

A new operation, `ReconcileFixedIncomeHolding(ctx, userID, id, confirmedInterestCentavos,
contributionCentavos int64)`, distinct from `UpdateFixedIncomeHolding`. `contributionCentavos`
may be zero (pure interest confirmation, no new money). On success:
`InvestedAmountCentavos += confirmedInterestCentavos + contributionCentavos`;
`TotalContributedCentavos += contributionCentavos` (interest never touches it — this is what
makes aporte vs. valorização unambiguous by construction, BR-1101); `LastReconciledAt = now`.
`FixedIncomeResponse` gains a computed, never-persisted `EstimatedInterestCentavos` — the current
`AccrueSimpleInterest` estimate since `LastReconciledAt` — so a client can pre-fill the
confirmation field with the system's best guess instead of an empty input.

#### Acceptance Criteria

- [ ] `contributionCentavos = 0` is valid and reconciles interest only.
- [ ] Both amounts must be `>= 0`; a negative value is rejected (`400`, mirrors `ErrNegativeAmount`).
- [ ] Ownership-scoped like every other holding mutation — `ErrHoldingNotFound` for a missing or
      unowned id (no cross-user existence oracle, SPEC-102 BR-1021).
- [ ] Reconciling twice in immediate succession is allowed (no idempotency lock needed — each
      call is an atomic, additive update); the second call's estimate baseline reflects the
      first's updated `LastReconciledAt`.
- [ ] `EstimatedInterestCentavos` uses the same `AccrueSimpleInterest` + `EffectiveAnnualRateBps`
      (SPEC-109 FR-1092) machinery already proven for the Dashboard, now keyed off
      `LastReconciledAt` instead of `CreatedAt`.

### FR-1104 — Dashboard growth splits by asset class

`internal/dashboard/compute.go`'s `ClassSlice` gains `InvestedCentavos int64`, `GrowthCentavos
int64`, `GrowthBps int`. FII growth is `fiiCurrent - fiiInvested` (cost basis unchanged — average
price × quantity, same as today). Fixed-income growth is `fiCurrent - fiTotalContributed`, using
the **new** `TotalContributedCentavos` as cost basis — not `InvestedAmountCentavos` — so a
reconciled contribution never shows up as growth. The FI accrual formula swaps its elapsed-time
anchor from `CreatedAt` to `LastReconciledAt`. The existing blended `Summary.GrowthCentavos`/
`GrowthBps` is kept, unchanged in meaning (sum of both classes) — nothing is removed from the API,
only added, so no existing consumer breaks.

#### Acceptance Criteria

- [ ] Immediately after a reconciliation with `contributionCentavos = 0`, the holding's growth
      (current − total contributed) equals exactly the confirmed interest — never resets to zero.
- [ ] Immediately after a reconciliation with `confirmedInterestCentavos = 0` and a positive
      `contributionCentavos`, growth is unchanged (a contribution is never growth) while
      `total_contributed_centavos` increases by the contribution.
- [ ] A `prefixado`-only, never-reconciled portfolio (pre-spec holdings) produces byte-for-byte
      identical `Summary.GrowthCentavos` to before this spec (regression-tested, mirrors SPEC-109
      BR-1093).
- [ ] Reconciliation (Σ per-class `GrowthCentavos` == `Summary.GrowthCentavos`) holds exactly
      (BR-1034 extended).

### FR-1105 — Portfolio "needs update" indicator

`Dashboard` gains `FixedIncomeReconciliationDue []string` (the names of fixed-income holdings
whose `LastReconciledAt` falls in a calendar month strictly before the current one, per the
injected `Clock`, UTC) and `NeedsAttention bool` — true when either
`FixedIncomeReconciliationDue` or the existing `StaleTickers` (FII, FR-1036) is non-empty. This
reuses the FII staleness concept (FR-1036) rather than inventing a parallel mechanism.

#### Acceptance Criteria

- [ ] A holding reconciled (or created) in the current calendar month is **not** due.
- [ ] A holding last reconciled in a prior calendar month — even one day into the new month — is
      due; this re-triggers automatically at the start of every new month with no stored/cron
      state (pure function of `LastReconciledAt` + `Clock.Now()`).
- [ ] An empty portfolio (no holdings at all) has `NeedsAttention = false`.
- [ ] `NeedsAttention` is a derived convenience field only — computed fresh on every read, never
      persisted (mirrors BR-1092's "derived fact, never conflated" posture).

### FR-1109 — Per-holding value breakdown

`Dashboard` gains two new lists: `FIIHoldings []FIIHoldingSlice{Ticker marketdata.Ticker,
ValueCentavos int64, ShareBps int}` (current value + share of the FII total per held ticker) and
`FixedIncomeHoldings []FixedIncomeHoldingSlice{ID, Name string, ValueCentavos, GrowthCentavos
int64, ShareBps int}` (current value, growth, and share of the FI total per holding). Both are
computed in `compute.go`'s existing per-holding loops (FR-1031/FR-1104) — no new query, just
appending an entry alongside the existing sector/class aggregation — and preserve the order the
holdings arrive in (`ORDER BY created_at`, the repository's existing deterministic order;
BR-1031), never re-sorted by value or name.

#### Acceptance Criteria

- [ ] `Σ FIIHoldings[].ValueCentavos == fii_current_value` and `Σ FixedIncomeHoldings[].ValueCentavos
      == fixed_income_current_value` (reconciles against the existing `ClassSlice`, BR-1034
      extended).
- [ ] `ShareBps` on each entry is that holding's share of its **own asset class** total (FII share
      of FII total; FI share of FI total) — not of the whole portfolio, matching how sector shares
      already work (FR-1033).
- [ ] A FII with no quote (stale, FR-1036) still appears, valued at cost basis, consistent with
      the existing stale-fallback behavior — it is not silently dropped from the list.
- [ ] `FixedIncomeHoldings[].GrowthCentavos` per entry is `ValueCentavos -
      TotalContributedCentavos` for that holding (mirrors FR-1104's class-level formula, at
      holding granularity); `Σ` reconciles to the class-level `GrowthCentavos`.
- [ ] Empty portfolio → both lists are empty arrays (never `null`), consistent with FR-1036's
      empty-portfolio posture.

### FR-1106 — API contract & money on the wire

`FixedIncomeResponse` gains `total_contributed_centavos`, `last_reconciled_at`,
`estimated_interest_centavos` (computed), `reconciliation_due` (computed, bool). A new endpoint
`POST /holdings/fixed-income/{id}/reconcile` accepts `{confirmed_interest_centavos,
contribution_centavos}` and returns the updated `FixedIncomeResponse`. The `Dashboard` schema's
allocation entries gain `invested_centavos`/`growth_centavos`/`growth_bps`; the top level gains
`needs_attention`/`fixed_income_reconciliation_due` and the new `fii_holdings`/
`fixed_income_holdings` per-holding arrays (FR-1109). Money as integer `*_centavos`, rates/shares
as integer `*_bps` — never a float, on every new field (BR-1022).

#### Acceptance Criteria

- [ ] `api/openapi.yaml` documents the new endpoint and every extended schema; the drift test
      (`openapi_test.go`) passes.
- [ ] The reconcile endpoint requires authentication (not on the public allowlist) and returns
      `{"error":"..."}` on failure, consistent with every other write endpoint.

### FR-1107 — Observability

The reconcile action gets its own named span (mirrors every other mutating endpoint's
`otelhttp` route span); logs carry `user_id` + `holding_id` + outcome, never money values.

#### Acceptance Criteria

- [ ] `POST /holdings/fixed-income/{id}/reconcile` inherits a route span; no PII/money on spans
      or logs.

### FR-1108 — Documentation

#### Acceptance Criteria

- [ ] `CHANGELOG.md` `[Unreleased]` updated in the same change as the implementation.
- [ ] `README` endpoint list updated if applicable.
- [ ] PT-BR lesson `docs/lessons/SPEC-110-aula.html` produced on close.

---

## 4. User Flows

### Main Flow (monthly reconciliation)

1. The user opens a fixed-income holding's edit surface (the existing SPEC-211 Carteira dialog,
   evolved per §15). It shows the system's `estimated_interest_centavos` since
   `last_reconciled_at`, pre-filled into a confirmation field, plus a contribution field
   defaulting to zero, and an explanation of what reconciliation is and why (§ Forward Note).
2. The user confirms or adjusts the interest figure, optionally enters a new contribution, and
   submits.
3. `POST /holdings/fixed-income/{id}/reconcile` updates the holding: balance grows by interest +
   contribution; lifetime contributed grows by the contribution only; the accrual clock resets.
4. The Dashboard's next read reflects the confirmed figures; the holding drops out of
   `fixed_income_reconciliation_due` for the current month.

### Alternative Flow — plain correction (no reconciliation)

1. The user edits a holding's name, rate, or a typo'd invested amount via the existing `PUT`.
2. If `invested_amount_centavos` changed, the accrual clock resets (FR-1102) but
   `total_contributed_centavos` is untouched — this is a correction, not new money.

### Alternative Flow — portfolio has stale holdings

1. The user opens the Dashboard. One FII has no fresh quote (existing `stale_tickers`, FR-1036)
   and one fixed-income holding was last reconciled two months ago.
2. `needs_attention = true`; `fixed_income_reconciliation_due` lists the stale holding by name.
3. The user reconciles it; on the next Dashboard read, `needs_attention` reflects only the
   still-stale FII (if any).

---

## 5. Business Rules

### BR-1101 — Aporte vs. valorização is unambiguous by construction

`TotalContributedCentavos` changes **only** via an explicit `contributionCentavos` at creation or
reconciliation — interest, confirmed or estimated, never touches it. This replaces "infer
contribution from a balance delta" (impossible to disambiguate, per the design discussion this
spec originates from) with "tag every balance change with its cause at the moment it happens."

### BR-1102 — Money and rates stay integer, including the reconciliation math

Mirrors BR-1022/SPEC-109 BR-1091: `total_contributed_centavos`, `estimated_interest_centavos`,
and every reconciliation amount are `int64` centavos; no float anywhere, including the wire.

### BR-1103 — Backward compatibility is non-negotiable

Every holding written before this spec ships defaults to `total_contributed_centavos =
invested_amount_centavos` and `last_reconciled_at = created_at`, producing byte-for-byte
identical Dashboard output to before this spec until its first reconciliation (mirrors SPEC-109
BR-1093).

### BR-1104 — Reconciliation is additive-only in the MVP

There is no undo/adjust endpoint for a mistaken reconciliation in this spec — a wrong entry is
fixed via the existing `PUT` correction path on `invested_amount_centavos` (which does not touch
`total_contributed_centavos`). A dedicated correction/undo flow is deferred (§14 Open Questions).

### BR-1105 — Reconciliation is per-holding, never portfolio-atomic

A stale or due holding never blocks reading, editing, or reconciling any other holding — the
Dashboard, the portfolio list, and reconciliation are each always available independently, per
holding. (Rejected alternative: a single all-or-nothing "update everything" screen — see §14 D3.)

### BR-1106 — Identity and ownership unchanged

Fixed-income holdings keep the existing SPEC-102 per-user ownership rules unchanged; the reconcile
action is scoped and gated exactly like `UpdateFixedIncomeHolding`/`DeleteFixedIncomeHolding`.

### BR-1107 — Per-holding figures always reconcile to their class total

`Σ FIIHoldings[].ValueCentavos` equals the FII `ClassSlice.ValueCentavos`; `Σ
FixedIncomeHoldings[].ValueCentavos` equals the Fixed Income `ClassSlice.ValueCentavos` (BR-1034
extended to holding granularity, FR-1109) — the per-holding breakdown is a strict decomposition of
the existing totals, never a separately-sourced figure that could drift from them.

---

## 6. Domain Model

### Entity: FixedIncomeHolding (extended)

| Field                    | Type      | Description                                                          |
| ------------------------ | --------- | ---------------------------------------------------------------------- |
| TotalContributedCentavos | int64     | New. Lifetime sum of money the user has contributed — the cost basis for growth. Grows only via an explicit contribution (create or reconcile). |
| LastReconciledAt         | time.Time | New. The accrual clock — replaces `CreatedAt` for interest-accrual purposes; reset by any change to `InvestedAmountCentavos` (edit or reconcile). |
| InvestedAmountCentavos   | int64     | Reinterpreted (existing column): the current principal/balance basis for accrual — grows via contributions **and** confirmed interest. No longer the cost basis by itself. |

### Computed (never persisted)

| Field                     | Description                                                                 |
| ------------------------- | ---------------------------------------------------------------------------- |
| EstimatedInterestCentavos | `AccrueSimpleInterest(InvestedAmountCentavos, EffectiveAnnualRateBps, days(LastReconciledAt, now))` — the pre-fill hint for reconciliation. |
| ReconciliationDue         | `LastReconciledAt`'s calendar month is strictly before the current one (Clock, UTC). |

### Dashboard `ClassSlice` (extended, SPEC-103)

Gains `InvestedCentavos int64`, `GrowthCentavos int64`, `GrowthBps int` alongside the existing
`Class`/`ValueCentavos`/`ShareBps`.

### Dashboard (extended, SPEC-103)

Gains `FixedIncomeReconciliationDue []string`, `NeedsAttention bool`, `FIIHoldings
[]FIIHoldingSlice`, and `FixedIncomeHoldings []FixedIncomeHoldingSlice` (FR-1109).

### Value objects: FIIHoldingSlice / FixedIncomeHoldingSlice (new, SPEC-103)

Mirror the existing `SectorSlice` shape (value + share of parent total). `FIIHoldingSlice`:
`Ticker`, `ValueCentavos`, `ShareBps`. `FixedIncomeHoldingSlice`: `ID`, `Name`, `ValueCentavos`,
`GrowthCentavos`, `ShareBps`. (`ID`+`Name` because, unlike a FII ticker, two fixed-income
holdings can share a display name — e.g. two CDBs both named "CDB Nubank" opened months apart.)

---

## 7. Ports (consumer-defined)

No new ports. `ReconcileFixedIncomeHolding` is added to the existing `portfolio.Repository`
(persistence write) and exposed through the existing `Service`, following the same shape as
`UpdateFixedIncomeHolding`. The Dashboard's existing `HoldingsReader`/`QuoteSource` ports (SPEC-103
§7) are unchanged — `LastReconciledAt`/`TotalContributedCentavos` simply ride along on
`portfolio.Holdings.FixedIncome` as extra fields.

---

## 8. API Contract

### `POST /holdings/fixed-income/{id}/reconcile` (new)

#### Request

```json
{
  "confirmed_interest_centavos": 963,
  "contribution_centavos": 0
}
```

#### Response

```json
{
  "id": "5c1e...",
  "name": "CDB Mercado Livre",
  "institution": "Mercado Livre",
  "invested_amount_centavos": 3128843,
  "total_contributed_centavos": 3127880,
  "annual_rate_bps": 10700,
  "indexer_type": "cdi_percentual",
  "effective_annual_rate_bps": 1124,
  "estimated_interest_centavos": 0,
  "reconciliation_due": false,
  "last_reconciled_at": "2026-07-15T02:16:18Z",
  "maturity_date": null,
  "liquidity_type": "daily",
  "created_at": "2026-07-13T22:51:27Z",
  "updated_at": "2026-07-15T02:16:18Z"
}
```

### `GET /dashboard` (extended)

Allocation entries gain `invested_centavos`/`growth_centavos`/`growth_bps`; the top level gains
`needs_attention` (bool), `fixed_income_reconciliation_due` (array of holding names), and the new
per-holding arrays:

```json
{
  "fii_holdings": [
    { "ticker": "HGLG11", "value_centavos": 432000, "share_bps": 950 }
  ],
  "fixed_income_holdings": [
    {
      "id": "5c1e...",
      "name": "CDB Mercado Livre",
      "value_centavos": 3128843,
      "growth_centavos": 963,
      "share_bps": 3510
    }
  ]
}
```

---

## 9. Data Model

### Migration (new, additive) — extends `fixed_income_holdings`

```sql
ALTER TABLE fixed_income_holdings
  ADD COLUMN total_contributed_centavos BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN last_reconciled_at TIMESTAMPTZ;

UPDATE fixed_income_holdings
  SET total_contributed_centavos = invested_amount_centavos,
      last_reconciled_at = created_at;

ALTER TABLE fixed_income_holdings
  ALTER COLUMN last_reconciled_at SET NOT NULL;
```

No new table — the reconcile action is an `UPDATE` on the existing row, mirroring
`UpdateFixedIncomeHolding`'s shape (SPEC-102). (A reconciliation history/audit ledger is
deliberately deferred — §14 Open Questions.)

### Indexes

None new — reads stay scoped by the existing `(id, user_id)` / `user_id` indexes (SPEC-002).

---

## 10. Edge Cases

| Scenario | Expected behavior |
| -------- | ----------------- |
| Reconciling the same day a holding was created | Valid; `estimated_interest_centavos` is near zero (tiny elapsed days), not an error. |
| Negative `confirmed_interest_centavos` or `contribution_centavos` | `400`, mirrors `ErrNegativeAmount`. |
| Reconciling twice in immediate succession | Both succeed; the second's estimate baseline reflects the first's updated `last_reconciled_at`. |
| A holding past `maturity_date` (`at_maturity`) | Reconciliation still allowed — no auto-close in MVP (mirrors SPEC-102's "past-maturity rule applies to creation only"). |
| Portfolio with zero holdings | `needs_attention = false`, `fixed_income_reconciliation_due = []`. |
| A user who never reconciles for many months | `estimated_interest_centavos` keeps accruing via simple interest over the full elapsed period — still labeled an estimate, never silently promoted to confirmed. |
| Editing only metadata (no `invested_amount_centavos` change) | `last_reconciled_at` untouched; no interest/contribution side effect. |

---

## 11. Security Requirements

### Authentication / Authorization

The reconcile action sits behind the same deny-by-default auth gate and is scoped/ownership-
checked exactly like `UpdateFixedIncomeHolding`/`DeleteFixedIncomeHolding` (SPEC-102, unchanged);
`ErrHoldingNotFound` for a missing or unowned id (no cross-user existence oracle, BR-1021).

### Data Protection

No new secrets or PII; the same money fields already governed by SPEC-102's per-user isolation.

---

## 12. Observability

- **Traces** — `otelhttp` route span for `POST /holdings/fixed-income/{id}/reconcile`; no new
  spans needed for the Dashboard's extended fields (same read path as SPEC-103).
- **Logs** — `user_id` + `holding_id` + outcome; never money values (mirrors SPEC-102/103).
- **Metrics** — none new required in MVP.

---

## 13. Testing Strategy

### Unit Tests

- `FixedIncomeHolding` bookkeeping (table-driven): create sets `TotalContributed =
  InvestedAmount`; plain edit changing `invested_amount_centavos` resets `LastReconciledAt` and
  leaves `TotalContributed` untouched; reconcile updates all three fields per FR-1103's formula;
  negative amounts rejected.
- Dashboard `compute.go`: per-class `GrowthCentavos`/`GrowthBps` reconciliation (Σ classes ==
  `Summary.GrowthCentavos`); FI growth uses `TotalContributedCentavos`, not
  `InvestedAmountCentavos`; accrual anchors off `LastReconciledAt`; `NeedsAttention`/
  `FixedIncomeReconciliationDue` with a fake `Clock` crossing a month boundary.
- `FIIHoldings`/`FixedIncomeHoldings` (FR-1109): per-holding values sum to their class total; a
  stale (quote-less) FII still appears at cost basis; empty portfolio yields empty (not `nil`)
  lists; deterministic ordering matches input order.
- Regression: a `prefixado`, never-reconciled portfolio produces byte-for-byte identical
  `Summary.GrowthCentavos` before/after this spec.

### Integration Tests (gated, `TEST_DATABASE_URL`)

- Migration up/down round-trip; existing rows backfill correctly.
- Real Postgres: create → reconcile (interest only) → reconcile (interest + contribution) →
  assert `invested_amount_centavos`/`total_contributed_centavos`/`last_reconciled_at` at each
  step; assert the Dashboard's per-class growth and `needs_attention` reflect it end-to-end.
- Regression: reproduce the pre-spec bug (edit bumping `invested_amount_centavos` inflating
  retroactive interest) against real Postgres, prove it fixed.

### Quality gate

`task vet`, `task test:short`, gofmt-clean; integration serialized when a DB is present.

---

## 14. Definition of Done

- [ ] FR-1101…FR-1109 implemented; BR-1101…BR-1107 respected; acceptance criteria met.
- [ ] Migration up/down proven against real Postgres; backward-compatibility regression green.
- [ ] `api/openapi.yaml` updated; drift test green.
- [ ] Dashboard (SPEC-103) consumes the split growth + staleness fields.
- [ ] `task vet` + `task test:short` clean; integration tests green against real Postgres.
- [ ] CHANGELOG updated; SPEC-110 + PLAN-110 flipped to Done; indexes updated.
- [ ] PT-BR lesson `docs/lessons/SPEC-110-aula.html` via **lesson-writer** (backend track).

---

## 15. Decisions (resolved)

| # | Decision | Recommendation |
| - | -------- | -------------- |
| D1 | Where to expose per-class growth | **Extend `ClassSlice`** (invested/growth alongside the existing value/share) over adding parallel ad-hoc fields to `Summary` — reuses the allocation structure the frontend already renders per class. |
| D2 | Cost basis for fixed-income growth | **`TotalContributedCentavos`**, not `InvestedAmountCentavos` — the latter now includes confirmed interest, so using it as cost basis would re-introduce the aporte/valorização conflation this spec exists to fix. |
| D3 | Reconciliation scope | **Per-holding**, not an atomic full-portfolio "update everything" screen (the user's original idea) — different institutions post interest on different days in real life; SPEC-103's FR-1036 already tolerates partial staleness per FII, this generalizes the same posture to FI. |
| D4 | Reconcile as a new endpoint vs. overloading `PUT` | **New `POST .../reconcile` endpoint** — keeps "correct a typo" (PUT) and "confirm this month's interest + report a contribution" (reconcile) semantically distinct, which is exactly the ambiguity that caused the original bug. |
| D5 | Pre-filled estimate vs. blank input for confirmed interest | **Pre-filled** with `EstimatedInterestCentavos` — the user confirms/corrects a computed fact rather than inventing one from scratch, in keeping with the project's "facts are computed" posture. Recommendation, not forced — see §16 Open Question 1. |

---

## 16. Open Questions (deferred, not blocking)

1. **Blank input vs. pre-filled estimate** (D5) — recommended pre-filled; confirm before PLAN-110.
2. **Reconciliation history/audit ledger** — not built here; `TotalContributedCentavos` +
   `LastReconciledAt` already disambiguate aporte from valorização without one. A "see my past
   reconciliations" view is a clean additive feature later if wanted.
3. **Correction/undo for a mistaken reconciliation** (BR-1104) — out of MVP scope; today's `PUT`
   correction path is the only fix available. Revisit if this proves painful in practice.
4. **Reminders beyond an in-app indicator** (email/push) — out of scope; no notification infra
   exists yet in this zero-cost MVP. `needs_attention` is an in-app, pull-based signal only.
5. **FII quote ingestion gap** (static `MARKETDATA_WATCHLIST` vs. holdings-driven discovery) is a
   separate, already-scoped future spec — deliberately not folded into this one.

---

## Forward note — frontend consequences (SPEC-211, SPEC-212, out of scope here)

This is a backend spec; two already-**Done** frontend specs need small follow-up revisions once
it ships (mirrors how SPEC-109 flagged SPEC-211 impact without blocking on it):

- **SPEC-212 (Painel/Dashboard)** — the blended "Valorização" currently shown at the
  total-patrimony hero should move into the FII and Fixed Income allocation cards individually
  (this spec's `ClassSlice.GrowthCentavos`); a "needs attention" banner should surface
  `needs_attention`/`fixed_income_reconciliation_due`; and two things should show real R$, not
  just `%`: (1) the asset-class `AllocationBar` legend already receives `value_centavos` from
  `GET /dashboard` today and simply never renders it (`allocation-sections.tsx` maps only `bps`
  into `AllocationSegment` — a pure wiring gap, no backend change needed for this part); (2) a new
  per-holding list/card for `fii_holdings`/`fixed_income_holdings` (FR-1109), showing each
  ticker's or holding's own R$ value, since none exists on the Painel today.
- **SPEC-211 (Carteira/Portfolio)** — the fixed-income edit dialog gains the reconciliation flow
  (§4 Main Flow): a pre-filled interest-confirmation field and a contribution field, wired to the
  new `POST .../reconcile` endpoint, alongside (not replacing) the existing plain-edit form. It
  must also surface a plain-language explanation of reconciliation — a modal, tooltip, or inline
  helper text (exact UI form is the SPEC-211 follow-up's call, not decided here), covering at
  least: (1) the pre-filled interest is a **system estimate** (the same simple-interest formula
  the Dashboard already uses), not verified bank data — the user should check it against their
  real statement before confirming; (2) why **aporte** (contribution) and **juros** (interest) are
  two separate fields — tagging each at the moment it happens is what makes "how much did I
  actually earn" answerable at all (BR-1101), instead of a guess from a balance delta; (3)
  skipping reconciliation isn't an error — the estimate just keeps accruing, unconfirmed, until
  the user catches up (no penalty, no data loss). This isn't gated by FR-013/FR-014 (no AI output
  here), but it's the same underlying principle: a number the user can't interpret correctly isn't
  actually useful, no matter how correctly it was computed. The explanation should be reachable
  every time reconciliation is used (a persistent affordance), not a one-shot onboarding tip a
  user who skips a few months would no longer see.

Both are small, additive frontend PLANs once SPEC-110 lands — not blocking this spec's approval.
