# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Dashboard (portfolio summary, allocation, sector exposure)   |
| Related Feature | SPEC-103 — the first read-across/compute feature; the patrimony view |
| Related Spec    | [SPEC-103](../02-specs/SPEC-103-dashboard.md)               |
| Version         | 0.1.0                                                        |
| Status          | Approved (decisions D1–D5 resolved)                          |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-26                                                   |

---

## 2. Objective

### Goal

Compute and serve the investor's dashboard — portfolio **summary** (total invested, current
estimated value = full patrimony, monthly passive income, growth; FR-004) and **allocation**
(by asset class + FII sector exposure; FR-005) — deterministically from holdings (SPEC-102) and
market data (SPEC-006), read-only.

### Expected Outcome

`GET /dashboard` returns the caller's summary + allocation + FII-sector breakdown (money as
integer centavos, shares as bps), reconciling exactly against the underlying holdings, with a
graceful cost-basis fallback for FIIs whose quote is missing. No new persistence.

---

## 3. Scope

### Included

- `internal/dashboard`: domain (`AssetClass`, `Summary`, `ClassSlice`, `SectorSlice`,
  `Dashboard`), the **pure deterministic computation**, the service, and the consumer ports
  (`HoldingsReader`, `QuoteSource`).
- `internal/platform/money` additions: a half-up **share-of-total → bps** helper (and a
  simple-interest **accrual** helper for FI current value, D2).
- HTTP `GET /dashboard` (auth-protected) + router/`cmd/api` wiring.
- Observability (inherited route span); tests; closeout.

### Excluded (SPEC-103 §2)

- AI insights / health score / rebalancing (SPEC-104/105/106) and **projections** (SPEC-107 —
  contribution plan + scenarios are captured as a forward note, not built here).
- Stocks/ETFs as managed classes (buckets present, always 0). Any write/persistence/migration.

---

## 4. Dependencies

### Technical Dependencies

- SPEC-102 `portfolio.Reader` (`ListHoldings`) + the `portfolio.Holdings`/`FIIHolding`/
  `FixedIncomeHolding` domain types.
- SPEC-006 marketdata quote repo (`GetFIIQuoteByTicker`) + `marketdata.FIIQuote`/`Ticker`/
  `Sector` types and `ErrFIIQuoteNotFound`.
- `internal/platform/money` (existing half-up `DecimalToMinor`; new ratio/accrual helpers),
  the `Clock` port, `auth.UserID(ctx)`, the `transport/http` router `Deps` + `writeJSON`.

### New Dependencies

- **None.** Pure stdlib + the existing stack.

### Blocking Decisions (SPEC-103 §14 — all resolved)

- **D1** new `internal/dashboard` package over `portfolio.Reader` + a quote port (acyclic) ·
  **D2** FI current value = simple-interest accrual to today · **D3** monthly income =
  Σ FII `last_dividend × quantity` · **D4** missing quote → cost-basis fallback + `stale_tickers`
  · **D5** all four allocation classes (Stocks/ETFs = 0). None blocking.
- **Minor (decide in Phase 2):** allocation/sector **bps rounding** — independent half-up per
  slice (bps may sum to 9999–10001 from rounding residue) vs largest-remainder (force Σ = 10000).
  Recommendation: **independent half-up** — the *values* reconcile exactly (the source of truth);
  bps are display. Documented.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/money` | Add `ShareBps(part, whole) int` (half-up) and a simple-interest accrual helper |
| `internal/transport/http` | New `dashboard.go` handler + `Deps.Dashboard`; register `GET /dashboard` (auto-protected) |
| `cmd/api` | Wire the dashboard service (portfolio reader + quote repo + clock) into `Deps` |
| `README` | Endpoints table gains `/dashboard` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/dashboard` | Domain + pure computation + service + consumer ports |

---

## 6. Implementation Strategy

### Approach

Bottom-up, with **determinism + reconciliation** as the throughline. The computation is a
**pure, side-effect-free function** of (holdings, quotes, now) → `Dashboard`: all sums/products
are `int64` centavos, all percentages integer **bps**, every division **half-up** via
`internal/platform/money`, so the same inputs always yield identical figures (PRD §6) and the
slices reconcile (Σ values = totals). The `Clock` drives the FI accrual and growth (no
`time.Now()`), keeping tests deterministic. Reads go through consumer-defined interfaces
(`HoldingsReader`, `QuoteSource`) so the core stays decoupled; identity is `auth.UserID(ctx)`.
Conventions: errors `%w` + sentinels; `ctx`-first; DTOs separate from domain (money on the wire
as integer `*_centavos`/`*_bps`, never float); **no package stutter** (`dashboard.Service`);
doc comments cite SPEC/BR; test files mirror source. No AI (FR-013/014 N/A).

### Rollout Method

Incremental, additive, read-only — a new auth-protected endpoint over existing data. No schema
change, nothing to migrate.

### Rollback Strategy

Remove the endpoint + wiring. No data or migration to revert. The money-helper additions are
pure and used only here.

---

## 7. Implementation Phases

### Phase 1 — Domain & Ports

#### Tasks

- [ ] `AssetClass` closed enum (`fii|fixed_income|stocks|etfs`); the computed structs
      (`Summary`, `ClassSlice`, `SectorSlice`, `Dashboard`) — money `int64` centavos, shares
      `int` bps; reuse `marketdata.Sector`/`Ticker`.
- [ ] Consumer ports `HoldingsReader` (`ListHoldings`) and `QuoteSource`
      (`GetFIIQuoteByTicker`), defined in `dashboard` (accept-interfaces).

#### Deliverables

- Compiling pure core (no SQL/HTTP); compiles against the existing portfolio/marketdata types.

---

### Phase 2 — Money helpers + the pure computation (the heart)

#### Tasks

- [ ] `internal/platform/money`: `ShareBps(part, whole int64) int` (half-up; whole == 0 → 0)
      and a simple-interest accrual helper (`invested × rate_bps × elapsedDays / (10000 × 365)`,
      half-up, overflow-safe). Unit-tested.
- [ ] `dashboard.Compute(holdings portfolio.Holdings, quotes map[Ticker]FIIQuote, now time.Time)
      (Dashboard, error)` — pure: per-FII current value (quote price × qty, or cost-basis
      fallback → `StaleTickers`), per-FI current value (cost basis + accrual to `now`), totals,
      monthly income (Σ FII last_dividend × qty), growth (centavos + bps), allocation by class,
      FII sector grouping. Deterministic; reconciling; divide-by-zero guarded.

#### Deliverables

- The pure computation + helpers, with table-driven tests proving **reconciliation,
  determinism, stale fallback, empty, and divide-by-zero guards** (no DB/HTTP needed).

---

### Phase 3 — Application (service)

#### Tasks

- [ ] `dashboard.Service`: `GetDashboard(ctx, userID) (Dashboard, error)` — read holdings via
      `HoldingsReader`; for each distinct held ticker fetch the quote via `QuoteSource`,
      mapping `marketdata.ErrFIIQuoteNotFound` to "stale" (skip, not an error); call `Compute`
      with the injected `Clock`'s now.
- [ ] Hand-written fakes (holdings reader + quote source, incl. a missing quote) for unit tests.

#### Deliverables

- Service that orchestrates reads + computation; service unit tests (happy, stale quote, empty).

---

### Phase 4 — API (transport)

#### Tasks

- [ ] `internal/transport/http/dashboard.go`: `GET /dashboard` → response DTO (money integer
      `*_centavos`, shares `*_bps`, `stale_tickers []string`), `writeJSON` envelope, identity
      from `auth.UserID(ctx)`; a service error → 500, empty portfolio → zeroed `200`.
- [ ] Add `Deps.Dashboard`; register the route (auto-protected); wire in `cmd/api`.

#### Deliverables

- Working endpoint behind auth; handler unit tests (money round-trip, empty `200`,
  identity-from-context, 401).

---

### Phase 5 — Observability

#### Tasks

- [ ] Confirm the `otelhttp` route span names `GET /dashboard`; the holdings/quote reads are
      child query spans (no argument values); assert **no money/figure values** on the span.

#### Deliverables

- Endpoint traced via the existing seam; a span-no-PII test.

---

### Phase 6 — Testing

#### Unit Tests

- [ ] Computation (table-driven: reconciliation, determinism, stale, empty, zero-guards, FI
      accrual via fake `Clock`); money helpers; service with fakes; handler.

#### Integration Tests (gated)

- [ ] Real Postgres (`TEST_DATABASE_URL`, `-p 1`): seed a user's holdings (SPEC-102 repo) +
      `fii_quotes` (SPEC-006 repo), call the service end-to-end, assert the figures reconcile.

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 7 — Documentation & Lesson

#### Tasks

- [ ] `README` endpoint + `CHANGELOG` entry.
- [ ] Flip SPEC-103 + PLAN-103 to **Done**; update indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-103-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Money rounding non-determinism / float creep (the core promise) | High | Pure integer math; every ratio half-up via `money`; table-driven determinism + reconciliation tests; no `float64` (domain, DTOs, DB read). |
| Allocation bps don't sum to exactly 10000 (independent rounding) | Low | Values reconcile exactly (source of truth); bps are display-rounded — documented; largest-remainder deferred unless the UI needs an exact 100%. |
| Integer overflow in FI accrual (`invested × rate × days`) | Medium | Accrual helper orders ops to avoid overflow (divide before multiply where safe) + guards; unit-tested with large values. |
| Missing/stale quotes corrupt the total | Medium | Cost-basis fallback keeps the total reconciling; ticker surfaced in `stale_tickers` (FR-1036); tested. |
| N quote queries per request (one per ticker) | Low | Fine for MVP holding counts; a batch quote read is a later optimization (open question). |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1031…FR-1039 implemented; BR-1031…BR-1038 respected; acceptance criteria met.
- [ ] Figures reconcile (Σ slices = totals); deterministic; stale fallback works; empty → zeros.

### Technical Validation

- [ ] Hexagonal (pure computation core; consumer ports; HTTP in transport); money int64
      centavos / bps end-to-end incl. the wire; `Clock` over `time.Now()`; identity from
      context; conventions (no stutter, `%w`, doc comments, test-file naming).

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
      hexagonal-reviewer + go-correctness-reviewer pass.

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] Computation proven deterministic + reconciling; stale fallback proven; gated integration
      green against real Postgres (seeded holdings + quotes).
- [ ] CHANGELOG + README updated; SPEC-103 + PLAN-103 → **Done**; indexes + `CLAUDE.md` status
      updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `internal/dashboard` (domain + computation + service + ports), `internal/platform/money`
  helpers, `internal/transport/http/dashboard.go`, `cmd/api` wiring.

### Infrastructure Deliverables

- None (no migration; read-only feature).

### Documentation Deliverables

- README endpoint, CHANGELOG entry, `SPEC-103-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch `dashboard.requests` (if added) once the UI consumes it.

### Future Improvements

- Reuse `dashboard.Compute` (or its parts) as the SPEC-104 Fact Builder core; a batch quote
  read; a portfolio freshness timestamp; largest-remainder bps if the UI needs exact 100%.

### Technical Debt

- FI accrual is simple-interest (A4 MVP approximation); revisit (compound / shared helper) with
  projections (SPEC-107).
