# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Portfolio Management (FII + Fixed Income holdings)           |
| Related Feature | SPEC-102 — the system of record for holdings; first feature with money |
| Related Spec    | [SPEC-102](../02-specs/SPEC-102-portfolio-management.md)     |
| Version         | 0.1.0                                                        |
| Status          | Approved (decisions D1–D5 resolved)                          |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-26                                                   |

---

## 2. Objective

### Goal

Deliver CRUD for FII holdings (FR-001) and Fixed Income holdings (FR-002) — per-user
isolated and ownership-enforced — and expose them via a `Reader` port for the dashboard,
Fact Builder, and projections.

### Expected Outcome

`POST/GET/PUT/DELETE /holdings/fii` and `/holdings/fixed-income` manage the caller's
holdings, money flowing as integer centavos end to end; `portfolio.Reader.ListHoldings`
serves the aggregate to SPEC-103/104/107. Nothing computes market value yet — this spec
stores the cost-basis facts.

---

## 3. Scope

### Included

- `internal/portfolio`: domain (`FIIHolding`, `FixedIncomeHolding`) + value objects
  (`Quantity`, `LiquidityType`; `Ticker` reused from `marketdata`, D1), sentinels, the
  service, the `Repository` port, and the consumer `Reader` port.
- `internal/portfolio/postgres` repository; migration `0005_holdings` (two tables).
- HTTP CRUD for both types (`/holdings/fii`, `/holdings/fixed-income`), behind the
  deny-by-default auth middleware; `cmd/api` wiring.
- Observability (inherited route spans); tests; closeout.

### Excluded (SPEC-102 §2)

- Current value / allocation / passive income / growth (SPEC-103/107 read holdings via the
  port).
- Stocks/ETFs as managed classes; transaction/lot history; brokerage import; any AI output.

---

## 4. Dependencies

### Technical Dependencies

- SPEC-003 (auth): `auth.UserID(ctx)`, deny-by-default middleware, `users` table (FK target).
- SPEC-002 (DB, migration runner). SPEC-004 (`otelhttp` route spans). `Clock` port.
- `internal/platform/money` (`DecimalToMinor`, half-up) for any inbound decimal parsing.
- **`marketdata.Ticker`** (D1) — the reused B3-ticker value object (feature → foundational
  seam, acyclic; mirrors `profile → auth`).
- `transport/http` router `Deps` + `writeJSON`/`decodeJSON` helpers.

### New Dependencies

- **None.** Pure stdlib + the existing stack.

### Blocking Decisions (SPEC-102 §14 — all resolved)

- **D1** reuse `marketdata.Ticker` · **D2** two tables · **D3** liquidity enum
  (`daily`|`at_maturity`, maturity required for at-maturity) · **D4** per-type routes +
  unified `Reader` · **D5** whole-number `Quantity` (>0). None blocking.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/transport/http` | New `holdings.go` handlers + `Deps.Portfolio`; register `/holdings/*` routes (auto-protected) |
| `cmd/api` | Wire the portfolio service (repo + clock) into `Deps` |
| `migrations/` | New `0005_holdings` up/down (manual) |
| `README` | Endpoints table gains `/holdings/*` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/portfolio` | Domain, value objects, sentinels, service, `Repository` + `Reader` ports |
| `internal/portfolio/postgres` | `Repository` adapter (scoped, ownership-checked SQL) |

---

## 6. Implementation Strategy

### Approach

Bottom-up and layered, with **money discipline and ownership as the two throughlines**.
Money is `int64` centavos / integer bps everywhere — domain, DB columns (`bigint`/`integer`),
and the JSON wire (DTOs expose `*_centavos`/`*_bps`, never floats; inbound decimals, if any,
parse via `internal/platform/money` half-up). Identity comes from `auth.UserID(ctx)`; reads
scope `WHERE user_id = $1` and mutations double-scope `WHERE id = $1 AND user_id = $2` so a
cross-user id is "not found", never an existence oracle. Conventions: `Quantity`/
`LiquidityType` parse-don't-validate value objects; `Ticker` reused from `marketdata`; errors
`%w` + sentinels; `Clock` over `time.Now()` (the at-maturity past-date rule uses it);
`ctx`-first; reads named `Get*By*`/`List*By*`; DTOs separate from domain; **no package-name
stutter** (`portfolio.Repository`, `portfolio.Reader`); test files mirror source. No AI here
(FR-013/014 N/A).

### Rollout Method

Incremental and additive — new endpoints behind existing auth; migration `0005` applied
manually with a tested down. No existing behavior changes.

### Rollback Strategy

Drop the endpoints / revert wiring; `0005` down removes both tables. No data migration.

---

## 7. Implementation Phases

### Phase 1 — Domain & Ports

#### Tasks

- [ ] Value objects: `Quantity` (whole number > 0) and `LiquidityType` (`daily`|`at_maturity`)
      with parse constructors + sentinels; reuse `marketdata.Ticker` for the FII ticker.
- [ ] Entities `FIIHolding` (ticker, quantity, average_price_centavos) and
      `FixedIncomeHolding` (name, institution, invested_amount_centavos, annual_rate_bps,
      maturity_date, liquidity_type) — money `int64` centavos, rate int bps; sentinels
      `ErrHoldingNotFound`, `ErrInvalidQuantity`, `ErrInvalidLiquidityType`,
      `ErrPastMaturity`, `ErrEmptyField`.
- [ ] Ports: `Repository` (Create/List/Update/Delete for each type, scoped) and the consumer
      `Reader` (`ListHoldings → Holdings{FII, FixedIncome}`).

#### Deliverables

- Compiling pure core (no SQL/HTTP); value-object unit tests green.

---

### Phase 2 — Persistence (migration + repository)

#### Tasks

- [ ] `migrations/0005_holdings.up.sql`/`.down.sql`: `fii_holdings` + `fixed_income_holdings`
      (UUID `id`, `user_id` FK `ON DELETE CASCADE`, `bigint` centavos / `integer` bps, no
      floats, index on `user_id`); tested down.
- [ ] `internal/portfolio/postgres`: per-type Create (RETURNING the row), `List…ByUserID`,
      Update (`WHERE id = $1 AND user_id = $2 … RETURNING`; no row → `ErrHoldingNotFound`),
      Delete (scoped; affected-rows 0 → `ErrHoldingNotFound`); the `Reader` aggregate;
      compile-time port assertion.

#### Deliverables

- Persistence with scoped, ownership-checked SQL; gated integration scaffold ready.

---

### Phase 3 — Application (service)

#### Tasks

- [ ] `portfolio.Service`: CRUD for both types (validate → build value objects → scoped
      repo call), the at-maturity past-date rule via the injected `Clock`, and the `Reader`
      implementation. Identity/`userID` always passed from the handler context.
- [ ] Hand-written fake repo for unit tests.

#### Deliverables

- Service with both CRUD sets + `Reader`; service unit tests (create→list→update→delete,
  not-found, validation, past-maturity) green.

---

### Phase 4 — API (transport)

#### Tasks

- [ ] `internal/transport/http/holdings.go`: handlers for `POST/GET/PUT/DELETE /holdings/fii`
      and `/holdings/fixed-income`; request/response DTOs separate from domain with **money as
      integer centavos** (`average_price_centavos`, `invested_amount_centavos`,
      `annual_rate_bps`); `writeJSON` envelope; **`user_id` from `auth.UserID(ctx)`**, never
      body/path; ownership errors → `404`.
- [ ] Add `Deps.Portfolio`; register routes (auto-protected); wire in `cmd/api`.

#### Deliverables

- Working endpoints behind auth; handler unit tests (validation, identity-from-context,
  money round-trip, ownership 404, 401) green.

---

### Phase 5 — Observability

#### Tasks

- [ ] Confirm the `otelhttp` route spans name `/holdings/*`; DB calls are child query spans
      (no argument values). Optional `portfolio.mutations` counter by `{type, op, outcome}`.
      No PII beyond `user_id`.

#### Deliverables

- Endpoints traced via the existing seam; no new telemetry config.

---

### Phase 6 — Testing

#### Unit Tests

- [ ] Value objects (`Quantity`, `LiquidityType`, past-maturity with fake `Clock`); service
      CRUD for both types; handlers (validation, identity-from-context, money-as-centavos,
      ownership 404, 401).

#### Integration Tests (gated)

- [ ] Real Postgres (`TEST_DATABASE_URL`, `-p 1`): CRUD round-trip, `0005` up/down,
      **per-user isolation**, **ownership-scoped update/delete** (B cannot touch A's row),
      cascade-on-user-delete.

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 7 — Documentation & Lesson

#### Tasks

- [ ] `README` endpoints + `CHANGELOG` entry.
- [ ] Flip SPEC-102 + PLAN-102 to **Done**; update indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-102-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Money precision/float creep (first money feature) | High | `int64` centavos + integer bps end to end incl. the wire; DTOs never use `float64`; integration asserts exact round-trip. |
| Identity/ownership leak — body/path `user_id` trusted, or update/delete not double-scoped | High | `auth.UserID(ctx)` only; mutations `WHERE id = $1 AND user_id = $2`; explicit tests that B cannot touch A's row and that a body/path id is ignored. |
| `marketdata.Ticker` reuse creates an awkward edge | Low | Feature→foundational, acyclic (mirrors `profile → auth`); pure value object, so `portfolio` core stays SQL/HTTP-free. |
| At-maturity past-date rule wrong / not Clock-driven | Medium | `Parse*`/service rule uses the injected `Clock`; unit-tested with a fixed clock; only enforced for `at_maturity`. |
| Two entity types double the CRUD surface (drift between them) | Medium | Shared validation helpers where sensible; table-driven tests cover both; the `Repository` keeps symmetric method names. |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1021…FR-1029 implemented; BR-1021…BR-1026 respected; acceptance criteria met.
- [ ] Per-user isolation + ownership scoping proven; money exact end to end.

### Technical Validation

- [ ] Hexagonal layering (domain pure; SQL in adapter; HTTP in transport); identity/ownership
      from context; money `int64` centavos incl. the wire; conventions (no stutter, `Clock`,
      `%w`, doc comments, test-file naming).

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
      hexagonal-reviewer + go-correctness-reviewer pass.

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] `0005_holdings` up/down + per-user isolation + ownership scoping proven against real Postgres.
- [ ] CHANGELOG + README updated; SPEC-102 + PLAN-102 → **Done**; indexes + `CLAUDE.md`
      status updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `internal/portfolio` (domain + value objects + service + ports), `internal/portfolio/postgres`,
  `internal/transport/http/holdings.go`, `cmd/api` wiring.

### Infrastructure Deliverables

- Migration `0005_holdings` (up/down).

### Documentation Deliverables

- README endpoints, CHANGELOG entry, `SPEC-102-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch `portfolio.mutations` (if added) once the dashboard consumes holdings.

### Future Improvements

- Combined `GET /holdings`; transaction/lot history + average-price recomputation; validating
  FII tickers against ingested market data; promoting `Ticker` to `internal/b3` if a 3rd
  consumer appears.

### Technical Debt

- Revisit the two-table CRUD symmetry if a third holding type is added.
