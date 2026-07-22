# PLAN-110 — Fixed-Income Reconciliation & Portfolio Update Indicator

## 1. Document Information

| Field           | Value                                   |
| --------------- | ---------------------------------------- |
| Plan Name       | Fixed-Income Reconciliation & Portfolio Update Indicator |
| Related Feature | Fixed-Income Reconciliation & Portfolio Update Indicator |
| Related Spec    | [SPEC-110](../02-specs/SPEC-110-fixed-income-reconciliation.md) (Approved) |
| Version         | 0.1.0                                      |
| Status          | Approved                                   |
| Author          | Gabigol                                   |
| Last Updated    | 2026-07-20                                 |

---

## 2. Objective

### Goal

Let a user confirm/adjust the interest their fixed-income holdings have earned and report new
contributions separately (so aporte and valorização are never conflated), fix the accrual-clock
bug where editing a holding's balance retroactively back-dates interest, and give the Dashboard a
per-class/per-holding growth breakdown plus a "needs your attention" signal.

### Expected Outcome

`POST /holdings/fixed-income/{id}/reconcile` lets the user confirm this month's interest and
report a contribution (zero allowed); `PUT` no longer produces incorrect interest when it touches
the invested amount; `GET /dashboard` shows FII vs. Fixed Income growth separately, a per-holding
R$ breakdown for both classes, and a `needs_attention` flag driven by stale FIIs and
never-reconciled fixed-income holdings. Every holding written before this plan ships is
byte-for-byte unaffected until its first reconciliation.

---

## 3. Scope

### Included

- `FixedIncomeHolding` gains `TotalContributedCentavos`, `LastReconciledAt` (persisted) and
  `EstimatedInterestCentavos`, `ReconciliationDue` (computed, never persisted — same treatment as
  SPEC-109's `EffectiveAnnualRateBps`).
- Migration `0008` (additive): the two new columns, backfilled from existing data.
- `Repository.ReconcileFixedIncomeHolding` (new) + `POST /holdings/fixed-income/{id}/reconcile`.
- The `PUT` accrual-clock bug fix (FR-1102), done as a single atomic `UPDATE`, not a
  read-then-write (race-free by construction).
- `internal/dashboard/compute.go`: per-class `InvestedCentavos`/`GrowthCentavos`/`GrowthBps` on
  `ClassSlice`; `FixedIncomeReconciliationDue`/`NeedsAttention` on `Dashboard`; the new
  `FIIHoldings`/`FixedIncomeHoldings` per-holding breakdown (FR-1109).
- `api/openapi.yaml`: the new endpoint + every extended schema.
- Unit + gated integration tests; CHANGELOG/README/PT-BR lesson.

### Excluded

- Any frontend change (SPEC-211/SPEC-212 follow-ups — see the SPEC's own Forward Note, tracked
  separately, not blocking this plan).
- A reconciliation history/audit ledger (SPEC-110 Open Question 2 — deliberately deferred).
- A correction/undo endpoint for a mistaken reconciliation (Open Question 3 — deferred; the
  existing `PUT` correction path is the only fix available in this plan).
- Email/push reminders (Open Question 4 — out of scope, no notification infra exists).
- The FII ticker-ingestion gap (SPEC-007 — already shipped, unrelated to this plan).

---

## 4. Dependencies

### Technical Dependencies

- `internal/portfolio` — `FixedIncomeHolding`, `Repository`, `Service` (`ResolveEffectiveRate`,
  `withEffectiveRate(s)`), the Postgres adapter (`CreateFixedIncomeHolding`/
  `UpdateFixedIncomeHolding` — the exact shape `ReconcileFixedIncomeHolding` mirrors).
- `internal/platform/money` — `AccrueSimpleInterest`, `ShareBps` (both already proven by SPEC-103/109).
- `internal/dashboard` — `compute.go`, `dashboard.go` (`ClassSlice`, `Dashboard`), `service.go`.
- `internal/transport/http` — `holdings.go` (handler/DTO pattern), `dashboard.go` (DTO pattern),
  `routes.go` (`routeTable`).

### External Dependencies

None new.

### Blocking Decisions

SPEC-110 §15 already resolves five design decisions (D1–D5) — this plan inherits them as-is. Two
items need resolving before/during this plan, not left to the SPEC's own text:

| # | Decision | Resolution |
|---|----------|------------|
| P1 | SPEC-110 is Draft, not Approved | Flip to **Approved** before `/spec-implement 110` starts Phase 1 (working agreement). |
| P2 | SPEC-110 Open Question 1 (D5: pre-filled vs. blank `confirmed_interest_centavos` on the client) | **Not blocking this plan.** The backend exposes `estimated_interest_centavos` on `FixedIncomeResponse` either way (FR-1103 AC5) — whether a future frontend pre-fills it or leaves it blank is a SPEC-211-follow-up UI decision, out of this backend plan's scope. |
| P3 | Where is `EstimatedInterestCentavos`/`ReconciliationDue` computed? | Same seam as `EffectiveAnnualRateBps` (SPEC-109): pure methods on `FixedIncomeHolding` taking `now time.Time` as a parameter (no I/O, `Clock`-free at the domain layer), invoked from `Service.withEffectiveRate(s)` using `s.clock.Now()` — extends the existing attach-computed-fields seam rather than inventing a second one. |
| P4 | How does `PUT` detect "did `invested_amount_centavos` change" without a race? | A **single atomic `UPDATE`** using a SQL `CASE` that compares the new value against the column's *current* (pre-update) value in the same statement — never a separate read-then-write (SQL, not application code, avoids a TOCTOU race between two concurrent edits). |

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/portfolio/holding.go` | `FixedIncomeHolding` gains 2 persisted + 2 computed fields; two new pure methods (`EstimateInterest(now)`, `IsReconciliationDue(now)`) |
| `internal/portfolio/ports.go` | `Repository` gains `ReconcileFixedIncomeHolding` |
| `internal/portfolio/service.go` | `Service` gains `ReconcileFixedIncomeHolding`; `withEffectiveRate(s)` also attaches the two new computed fields |
| `internal/portfolio/postgres/postgres.go` | `CreateFixedIncomeHolding` sets the two new columns; `UpdateFixedIncomeHolding`'s SQL gains the `CASE`-based clock reset (P4); new `ReconcileFixedIncomeHolding` method; `rebuildFixedIncome` reads the two new columns |
| `internal/dashboard/dashboard.go` | `ClassSlice` gains 3 fields; `Dashboard` gains 4 fields; 2 new slice types |
| `internal/dashboard/compute.go` | Track FI cost basis via `TotalContributedCentavos`; FI accrual anchors off `LastReconciledAt`; per-class growth; per-holding breakdown; `NeedsAttention` |
| `internal/transport/http/holdings.go` | `fixedIncomeResponse` gains 4 fields; new reconcile request/response DTOs + handler |
| `internal/transport/http/dashboard.go` | DTOs extended to match `dashboard.go`'s new fields |
| `internal/transport/http/routes.go` | New route: `POST /holdings/fixed-income/{id}/reconcile` |
| `api/openapi.yaml` | Extended `FixedIncomeResponse`; new path + request schema; extended `DashboardResponse` |
| `migrations/` | New `0008_fixed_income_reconciliation.{up,down}.sql` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `FixedIncomeHolding.EstimateInterest(now)` / `.IsReconciliationDue(now)` | Pure computed-field methods (P3), mirror `ResolveEffectiveRate`'s shape |
| `Repository.ReconcileFixedIncomeHolding` | Atomic, additive update: `invested_amount_centavos += confirmed+contribution`, `total_contributed_centavos += contribution`, `last_reconciled_at = now` |
| `POST /holdings/fixed-income/{id}/reconcile` handler | Mirrors `updateFixedIncomeHolding`'s shape (`internal/transport/http/holdings.go`) |
| `dashboard.FIIHoldingSlice` / `FixedIncomeHoldingSlice` | New value types (FR-1109), mirror `SectorSlice`'s value+share shape |

---

## 6. Implementation Strategy

### Approach

Bottom-up: domain fields/methods → migration → repository (including the P4 race-free `PUT` fix)
→ service → dashboard compute → HTTP + OpenAPI → tests → docs. Each phase keeps `task vet` +
`task test:short` green. The FR-1102 regression test (today's retroactive-interest bug) is
written *before* the fix, confirmed failing against the old code path, then confirmed fixed —
proving the fix actually addresses the diagnosed bug, not just plausible-looking code.

### Rollout Method

**Incremental**, backward-compatible migration (additive, defaults/backfills existing rows to
today's exact behavior — mirrors SPEC-109's BR-1093 precedent). No flag needed: the new
reconcile endpoint is simply unused until a client (a future SPEC-211 follow-up) calls it.

### Rollback Strategy

The migration's `down` drops both new columns. Every code path that reads
`TotalContributedCentavos`/`LastReconciledAt` only does so via the Go struct fields (backfilled on
migrate-up), so a rollback is safe at any point — no code depends on the columns existing beyond
this feature's own reads/writes.

---

## 7. Implementation Phases

### Phase 1 — Domain Layer

#### Tasks

- [ ] `internal/portfolio/holding.go`: add `TotalContributedCentavos int64`, `LastReconciledAt
      time.Time` (persisted) and `EstimatedInterestCentavos int64`, `ReconciliationDue bool`
      (computed, never persisted — doc comment mirrors `EffectiveAnnualRateBps`'s, citing
      FR-1101/BR-1101) to `FixedIncomeHolding`.
- [ ] Add `EstimateInterest(now time.Time) int64` — pure, calls the existing
      `money.AccrueSimpleInterest(InvestedAmountCentavos, EffectiveAnnualRateBps,
      daysBetween(LastReconciledAt, now))` (reuses `dashboard`'s `daysBetween` logic — either
      export it or duplicate the 4-line helper locally; decide during implementation, favor
      duplication over a cross-package export if it keeps `portfolio` from depending on
      `dashboard`, which would invert the existing dependency direction).
- [ ] Add `IsReconciliationDue(now time.Time) bool` — true iff `LastReconciledAt`'s
      `(year, month)` is strictly before `now`'s `(year, month)`, both UTC.
- [ ] `ErrNegativeAmount` (existing sentinel) is reused for reconciliation validation — no new
      sentinel needed.

#### Deliverables

- [ ] Table-driven unit tests (`holding_test.go`): `EstimateInterest` at zero/positive elapsed
      days; `IsReconciliationDue` — same month (false), prior month even by one day (true), a
      holding reconciled today (false).

---

### Phase 2 — Persistence Layer

#### Tasks

- [ ] `migrations/0008_fixed_income_reconciliation.up.sql`:
      ```sql
      ALTER TABLE fixed_income_holdings
        ADD COLUMN total_contributed_centavos BIGINT NOT NULL DEFAULT 0,
        ADD COLUMN last_reconciled_at TIMESTAMPTZ;
      UPDATE fixed_income_holdings
        SET total_contributed_centavos = invested_amount_centavos,
            last_reconciled_at = created_at;
      ALTER TABLE fixed_income_holdings ALTER COLUMN last_reconciled_at SET NOT NULL;
      ```
      `.down.sql` drops both columns.
- [ ] `internal/portfolio/postgres/postgres.go`: `CreateFixedIncomeHolding`'s `INSERT` sets
      `total_contributed_centavos = invested_amount_centavos`,
      `last_reconciled_at = created_at` (the opening balance is the first contribution, FR-1101
      AC2); `rebuildFixedIncome` reads both new columns.
- [ ] `UpdateFixedIncomeHolding`'s `SET` clause (P4, race-free): reset the accrual clock in the
      *same* statement using the pre-update value —
      ```sql
      UPDATE fixed_income_holdings
      SET name = $3, institution = $4, invested_amount_centavos = $5, annual_rate_bps = $6,
          indexer_type = $7, maturity_date = $8, liquidity_type = $9, updated_at = $10,
          last_reconciled_at = CASE WHEN invested_amount_centavos != $5 THEN $10
                                     ELSE last_reconciled_at END
      WHERE id = $1::uuid AND user_id = $2::uuid
      RETURNING created_at, updated_at, total_contributed_centavos, last_reconciled_at
      ```
      `total_contributed_centavos` is **not** in the `SET` list — a plain edit never touches it
      (FR-1102 AC1/BR-1101).
- [ ] New `ReconcileFixedIncomeHolding(ctx, userID, id string, confirmedInterestCentavos,
      contributionCentavos int64, now time.Time) (portfolio.FixedIncomeHolding, error)` —
      additive, atomic:
      ```sql
      UPDATE fixed_income_holdings
      SET invested_amount_centavos = invested_amount_centavos + $3 + $4,
          total_contributed_centavos = total_contributed_centavos + $4,
          last_reconciled_at = $5, updated_at = $5
      WHERE id = $1::uuid AND user_id = $2::uuid
      RETURNING invested_amount_centavos, total_contributed_centavos, ... (full row)
      ```
      `ErrHoldingNotFound` on no match (mirrors `notFound(err)`/`RowsAffected` conventions
      already used elsewhere in this file).

#### Deliverables

- [ ] Migration up/down round-trip integration test: existing rows backfill correctly.
- [ ] Gated integration test reproducing the **pre-fix bug** against real Postgres (create a
      holding, wait/simulate elapsed time, `PUT` with a bumped `invested_amount_centavos`, assert
      `last_reconciled_at` — proves both the bug's old shape and the fix, per the plan's Approach).
- [ ] Gated integration test: reconcile (interest-only) → reconcile (interest + contribution) →
      assert `invested_amount_centavos`/`total_contributed_centavos`/`last_reconciled_at` at each
      step, and that a concurrent-looking second reconcile still adds correctly (no lost update).

---

### Phase 3 — Application Layer

#### Tasks

- [ ] `internal/portfolio/ports.go`: `Repository` gains
      `ReconcileFixedIncomeHolding(ctx, userID, id string, confirmedInterestCentavos,
      contributionCentavos int64, now time.Time) (FixedIncomeHolding, error)`.
- [ ] `internal/portfolio/service.go`: new `Service.ReconcileFixedIncomeHolding(ctx, userID, id
      string, confirmedInterestCentavos, contributionCentavos int64) (FixedIncomeHolding,
      error)` — validates both amounts `>= 0` (`ErrNegativeAmount` on violation, FR-1103 AC2),
      calls `s.repo.ReconcileFixedIncomeHolding(..., s.clock.Now())`, returns
      `s.withEffectiveRate(ctx, updated)`.
- [ ] `withEffectiveRate`/`withEffectiveRates` (existing) also set `EstimatedInterestCentavos`
      (`h.EstimateInterest(s.clock.Now())`) and `ReconciliationDue`
      (`h.IsReconciliationDue(s.clock.Now())`) — one `Now()` call shared across all three
      computed fields per holding, consistent with the existing "one macro snapshot per
      request/response" posture.
- [ ] `internal/dashboard/compute.go`: split `totalInvested` into `fiiInvested`/`fiInvested`
      (currently combined); FI accrual anchors `daysBetween` off `h.LastReconciledAt` (not
      `h.CreatedAt`); FI cost basis for growth uses `h.TotalContributedCentavos`, accumulated as
      `fiTotalContributed`, separate from `fiInvested` (which still feeds `Summary.TotalInvestedCentavos`
      — unchanged meaning, still cost-basis-of-record, FR-1104 keeps `Summary.GrowthCentavos`
      backward-compatible).
- [ ] `compute.go`: `allocation(...)` extended to also return per-class
      `InvestedCentavos`/`GrowthCentavos`/`GrowthBps` (FII growth = `fiiCurrent - fiiInvested`;
      FI growth = `fiCurrent - fiTotalContributed`; Stocks/ETFs stay all-zero).
- [ ] `compute.go`: new `fiiHoldings(...)`/`fixedIncomeHoldings(...)` helpers, called from the
      existing per-holding loops (FR-1109) — append an entry per holding in input order (no
      re-sort), `ShareBps` relative to that class's own current-value total (mirrors `sectors()`'s
      share-of-`fiiTotal` pattern, not share-of-portfolio).
- [ ] `compute.go`: `FixedIncomeReconciliationDue`/`NeedsAttention` — collect FI holding names
      where `h.IsReconciliationDue(now)` (recomputed here too, since `Compute` is pure and
      doesn't reuse the Service-attached field — `Compute` receives `portfolio.Holdings` which
      already carries `ReconciliationDue` from `ListHoldings`'s `withEffectiveRates`, so it CAN
      reuse it directly instead of recomputing — decide during implementation which is cleaner;
      reusing avoids a second `now`-dependent code path computing the same fact twice).

#### Deliverables

- [ ] Unit tests (`portfolio/service_test.go`, hand-written fakes): `ReconcileFixedIncomeHolding`
      updates all three fields per the documented formula; negative amounts rejected; ownership
      scoping (`ErrHoldingNotFound` for an unowned id) via the existing fake repository pattern.
- [ ] `dashboard/compute_test.go`: per-class growth reconciliation (Σ classes ==
      `Summary.GrowthCentavos`); FI growth uses `TotalContributedCentavos` not
      `InvestedAmountCentavos`; `NeedsAttention`/`FixedIncomeReconciliationDue` with a fake
      `Clock` crossing a month boundary; `FIIHoldings`/`FixedIncomeHoldings` sum to their class
      totals, empty-portfolio yields empty (not `nil`) lists; **regression**: a `prefixado`,
      never-reconciled fixture produces byte-for-byte identical `Summary.GrowthCentavos` to a
      captured pre-SPEC-110 baseline.

---

### Phase 4 — API Layer

#### Tasks

- [ ] `internal/transport/http/holdings.go`: extend `fixedIncomeResponse` with
      `total_contributed_centavos`, `last_reconciled_at`, `estimated_interest_centavos`,
      `reconciliation_due`; `toFixedIncomeResponse` maps them.
- [ ] New `reconcileFixedIncomeRequest{ConfirmedInterestCentavos, ContributionCentavos int64}` +
      `reconcileFixedIncomeHolding` handler (mirrors `updateFixedIncomeHolding`'s shape: auth →
      decode → call service → `writeHoldingError` → `writeJSON(200, toFixedIncomeResponse(...))`).
      `PortfolioService` interface gains `ReconcileFixedIncomeHolding`.
- [ ] `internal/transport/http/routes.go`: register `POST
      /holdings/fixed-income/{id}/reconcile` → `holdingsH.reconcileFixedIncomeHolding`.
- [ ] `internal/transport/http/dashboard.go`: `classSliceResponse` gains
      `invested_centavos`/`growth_centavos`/`growth_bps`; `dashboardResponse` gains
      `needs_attention`, `fixed_income_reconciliation_due` (`[]string`, empty not `null`), and
      new `fii_holdings`/`fixed_income_holdings` (`[]fiiHoldingSliceResponse`/
      `[]fixedIncomeHoldingSliceResponse`, both new DTO types mirroring `sectorSliceResponse`'s
      shape); `toDashboardResponse` maps all of it.
- [ ] `api/openapi.yaml`: extend `FixedIncomeResponse`; add
      `POST /holdings/fixed-income/{id}/reconcile` (request schema
      `FixedIncomeReconcileRequest`, response `FixedIncomeResponse`, `400`/`401`/`404`
      responses matching the `PUT` endpoint's); extend `DashboardResponse`'s `allocation` items
      and top level per the SPEC's §8 examples.

#### Deliverables

- [ ] Handler tests (`holdings_test.go`): reconcile happy path, negative-amount `400`, unowned
      `404`, money-as-integer-centavos round-trip. `dashboard_test.go`: extended fields present
      and correctly mapped, empty portfolio → empty arrays not `null`.
- [ ] `openapi_test.go` drift test green (`TestOpenAPI_DocumentsEveryRoute` +
      `TestOpenAPI_NoStaleDocumentedRoutes`).

---

### Phase 5 — Observability

#### Tasks

- [ ] Confirm (no new code expected) the reconcile route inherits the standard `otelhttp`
      route-named span, same as every other mutating holdings endpoint.
- [ ] Confirm no money value is ever logged on the reconcile path (mirrors SPEC-102/103's
      no-PII/no-money-on-spans-or-logs posture) — a code-review check, not a new test.

#### Deliverables

- [ ] A short span-presence test for the new route (mirrors SPEC-109's
      `TestHTTP_MarketIndicatorsSpanRouteNamed` precedent), if the existing holdings span tests
      don't already generically cover any new `routeTable` entry.

---

### Phase 6 — Testing

#### Unit Tests

- Covered per-phase above (Phase 1 domain methods, Phase 3 service + compute, Phase 4 handlers).
- Full regression suite: a `prefixado`-only, never-reconciled portfolio's Dashboard/Projections
  output is byte-for-byte unchanged before/after this plan (BR-1103).

#### Integration Tests (gated, `TEST_DATABASE_URL` — a **disposable** Postgres, never the dev
compose DB on port 5433, per CLAUDE.md's rule and `.claude/hooks/block-dev-db-test.ps1`)

- Migration up/down round-trip.
- The pre-fix-bug reproduction (Phase 2).
- End-to-end: create → reconcile → `GET /dashboard` reflects the new per-class growth,
  per-holding breakdown, and `needs_attention` correctly.

#### Deliverables

- [ ] `task vet` + `task test:short` clean; full `go test ./... -count=1` green against a
      disposable Postgres (port 5434, matching the SPEC-007-established convention).
- [ ] `api/openapi.yaml` drift test green.

---

### Phase 7 — Documentation

#### Tasks

- [ ] `CHANGELOG.md` `[Unreleased]` updated.
- [ ] `README.md` endpoint list updated (new `POST .../reconcile` route).
- [ ] Flip **SPEC-110 + PLAN-110 → Done**; update `docs/02-specs/README.md` and
      `docs/03-plans/README.md`.
- [ ] PT-BR lesson `docs/lessons/SPEC-110-aula.html` via **lesson-writer** (backend track).
- [ ] Note in the closeout that the SPEC's own Forward Note (SPEC-211/SPEC-212 frontend
      follow-ups) is now unblocked — not built here, tracked separately.

#### Deliverables

- [ ] Docs updated, spec + plan closed, lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| The `PUT` accrual-clock fix (P4) is subtly wrong (e.g. compares against the new value instead of the old one) | High | The regression test is written to reproduce the **original bug** first, against a code path that doesn't yet have the fix, then re-run after — proves the fix addresses the actual diagnosed problem, not just a plausible-looking `CASE` expression |
| Reconciling concurrently (rare in a single-user MVP, but possible) causes a lost update | Medium | The `ReconcileFixedIncomeHolding` SQL is a single atomic `UPDATE ... SET x = x + $n` — Postgres row-level locking makes concurrent increments correct by construction, no read-then-write gap |
| Dashboard `Compute`'s purity/determinism guarantee breaks if `NeedsAttention`/`ReconciliationDue` end up computed two different ways in two places | Low | Phase 3 explicitly resolves this: `Compute` reuses the already-attached `ReconciliationDue` field from `Holdings.FixedIncome` rather than recomputing it independently |
| Silent regression in `prefixado`, never-reconciled Dashboard output | High | Byte-for-byte regression test against a captured pre-SPEC-110 baseline (mirrors SPEC-109's own precedent) |
| Forgetting the "disposable Postgres only" rule for integration tests | High (data loss, per the SPEC-007 incident) | `.claude/hooks/block-dev-db-test.ps1` already blocks `TEST_DATABASE_URL` pointed at port 5433; this plan's Phase 2/6 integration tests explicitly call out port 5434 |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1101…FR-1109 implemented; SPEC-110 acceptance criteria satisfied.
- [ ] BR-1101…BR-1107 respected (aporte/valorização separation, integer money, backward compat,
      additive-only reconciliation, per-holding non-atomic reconciliation, unchanged ownership,
      per-holding figures reconcile to class totals).

### Technical Validation

- [ ] `api/openapi.yaml` updated; drift test green.
- [ ] Money/rate stays `int64` centavos / integer bps everywhere in the new code.
- [ ] `Compute` remains pure (no I/O, deterministic) — the new fields are derived from already-
      resolved input, not fetched inside `Compute` itself.

### Quality Validation

- [ ] `task vet` + `task test:short` clean.
- [ ] `task test:integration` green against a **disposable** Postgres (never port 5433).
- [ ] Reviewed by **hexagonal-reviewer** + **go-correctness-reviewer**; blocking findings fixed.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-110 acceptance criteria satisfied.
- [ ] Migration `0008` up/down proven against real Postgres; backward-compatibility regression
      green (byte-for-byte `prefixado`, never-reconciled output).
- [ ] `api/openapi.yaml` updated; drift test green; no other endpoint's contract changed.
- [ ] `task vet`, `task test:short`, `go build ./...` clean; `task test:integration` green.
- [ ] CHANGELOG + README updated; SPEC-110 + PLAN-110 flipped to Done; indexes updated.
- [ ] PT-BR lesson `docs/lessons/SPEC-110-aula.html` produced.
- [ ] Reviewed by the backend review agents (hexagonal-reviewer, go-correctness-reviewer).

---

## 11. Deliverables

### Code Deliverables

- `FixedIncomeHolding` extensions + 2 new pure methods; `Repository.ReconcileFixedIncomeHolding`
  + the race-free `UpdateFixedIncomeHolding` fix; `Service.ReconcileFixedIncomeHolding`;
  `compute.go`'s per-class/per-holding growth + staleness; the new HTTP endpoint + extended
  Dashboard/FixedIncome DTOs; migration `0008`.

### Documentation Deliverables

- CHANGELOG entry, PT-BR lesson, `api/openapi.yaml` updates, specs/plans index updates.

---

## 12. Post-Implementation Tasks

### Future Improvements

- SPEC-211's fixed-income edit dialog gains the reconciliation flow (pre-filled interest
  confirmation + contribution field) plus a plain-language explanation of what reconciliation is
  and why aporte/juros are tracked separately (modal, tooltip, or inline helper text — SPEC-110's
  own Forward Note spells out the required content), wired to the new endpoint.
- SPEC-212's Painel moves "Valorização" into the per-class cards and adds a "needs attention"
  banner + per-holding R$ cards (same Forward Note).
- A reconciliation history/audit ledger, if a "see my past reconciliations" view is wanted later
  (Open Question 2).
- A correction/undo endpoint for a mistaken reconciliation, if the `PUT`-based workaround proves
  painful in practice (Open Question 3).

### Technical Debt

None anticipated — the migration is additive, the `PUT` fix is a same-statement `CASE` (no
follow-up cleanup needed), and the reconcile endpoint's atomicity means no locking/retry logic to
maintain.
