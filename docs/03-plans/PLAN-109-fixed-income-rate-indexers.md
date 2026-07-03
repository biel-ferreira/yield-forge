# PLAN-109 — Fixed-Income Rate Indexers (% do CDI / IPCA+)

## 1. Document Information

| Field           | Value                                   |
| --------------- | ---------------------------------------- |
| Plan Name       | Fixed-Income Rate Indexers                |
| Related Feature | Fixed-Income Rate Indexers                |
| Related Spec    | [SPEC-109](../02-specs/SPEC-109-fixed-income-rate-indexers.md) (Approved) |
| Version         | 0.1.0                                      |
| Status          | Done                                       |
| Author          | Gabigol                                   |
| Last Updated    | 2026-07-02                                |

---

## 2. Objective

### Goal

Add a rate **indexer** (`prefixado` | `cdi_percentual` | `ipca_spread`) to fixed-income holdings,
resolve the **effective current annual rate** at read-time from SPEC-006's already-ingested
SELIC/CDI/IPCA, wire that resolution into the Dashboard (SPEC-103) and Projections (SPEC-107), and
expose a new read-only `GET /market/indicators` endpoint.

### Expected Outcome

A `cdi_percentual` or `ipca_spread` holding's Dashboard current-value and Projections income always
reflect today's reference rate — no user action needed when SELIC moves. Every holding that existed
before this plan defaults to `prefixado` and produces byte-for-byte identical output.

---

## 3. Scope

### Included
- `Indexer` value object + `IndexerType` field on `FixedIncomeHolding` (`internal/portfolio`).
- A pure effective-rate resolution function, given an already-fetched macro reading.
- A `MacroReader` consumer port + wiring in `internal/dashboard` and `internal/projection`
  (mirroring the existing pattern in `internal/health` and `internal/insight/engine`).
- `GET /market/indicators` (new endpoint, reusing the existing `marketdata.MacroRepository`).
- Migration `0007` (additive), `api/openapi.yaml` updates, the SDD closeout.

### Excluded
- Any frontend change (SPEC-211, blocked on this plan, picks it up separately).
- New market-data ingestion — SPEC-006 already fetches SELIC/CDI/IPCA; this plan only *reads* it.
- Any change to FII holdings, liquidity type, or maturity-date rules (untouched).

---

## 4. Dependencies

### Technical Dependencies
- `internal/marketdata` — `MacroRepository.GetLatestMacroIndicator` (SPEC-006), already implemented
  and ingesting SELIC/CDI/IPCA.
- `internal/portfolio` — `FixedIncomeHolding`, `LiquidityType` (the value-object pattern this
  mirrors), the Postgres repository (SPEC-102).
- `internal/dashboard` (`compute.go:55`, `money.AccrueSimpleInterest`) and `internal/projection`
  (FI income calc) — both consume the resolved rate.

### External Dependencies
None new.

### Blocking Decisions

| # | Decision | Resolution (this plan) |
|---|----------|------------------------|
| D1 | Where does the resolution function live? | `internal/portfolio` (a method on `FixedIncomeHolding`, e.g. `EffectiveAnnualRateBps(macro map[marketdata.Indicator]marketdata.MacroIndicator) (int64, error)`) — colocated with the domain type it resolves, pure (no I/O), taking already-fetched readings. Portfolio already legitimately imports `marketdata` for `Ticker` (SPEC-102 D1), so this is not a new layering exception. |
| D2 | Who fetches the macro readings (the I/O)? | Each **service** that needs the resolved rate (`dashboard.Service`, `projection.Service`, and `portfolio`'s own read path for `GET /holdings/fixed-income`) gets its **own small `MacroReader` port**, mirroring the existing pattern in `internal/health/ports.go` and `internal/insight/engine/ports.go` — all three satisfied by the same `marketdata/postgres.MacroRepository` adapter; no new adapter code. |
| D3 | Degradation on `ErrMacroNotFound` (SPEC-109 Open Question #2) | **Resolved:** fall back to `effective = the raw stored annual_rate_bps` (never null, never a crash, never a silent zero) — the number is only "wrong" (reads as if prefixado) in a freshly-provisioned environment before the first ingestion run, and self-heals the moment ingestion runs once. Documented on the resolution function's doc comment. |
| D4 | Is `Compute` (dashboard/projection) still pure? | **Yes.** The resolution (I/O via `MacroReader`) happens in the **service** layer, *before* calling the existing pure `Compute`/projection functions — those functions receive an already-resolved effective rate per FI holding, same shape as today, so their purity/determinism guarantee is unchanged. |
| D5 | New migration number | `0007_fixed_income_indexer` (next free after `0006_chat`). |

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/portfolio/{fixedincome.go or similar, postgres/postgres.go}` | Add `IndexerType` field + read/write mapping |
| `internal/dashboard/{ports.go, service.go, compute.go}` | New `MacroReader` port; service resolves effective rate before `Compute` |
| `internal/projection/{ports.go, service.go, compute.go or facts.go}` | Same as dashboard, for FI income |
| `internal/transport/http/routes.go` | Register `GET /market/indicators`; extend FI holding DTOs |
| `migrations/` | New `0007_fixed_income_indexer.{up,down}.sql` |
| `api/openapi.yaml` | Extend `FixedIncomeRequest/Response`; add `MarketIndicatorResponse` + the new path |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/portfolio/indexer.go` | `Indexer` value object (`ParseIndexer`, closed enum) — mirrors `liquiditytype.go` |
| `(method) FixedIncomeHolding.EffectiveAnnualRateBps` | Pure resolution function (D1) |
| `internal/dashboard` `MacroReader`, `internal/projection` `MacroReader` | Consumer-defined ports (D2) |
| `internal/marketdata` HTTP handler for `GET /market/indicators` (or a small new `internal/market` read-only feature slice — TBD in Phase 4, whichever keeps the marketdata core free of HTTP per the layering rule) | Serves the new endpoint |

---

## 6. Implementation Strategy

### Approach
Bottom-up: domain value object → persistence → the three consuming services (portfolio's own
read path, dashboard, projection) → the new endpoint → tests → docs. Each phase keeps
`go vet ./...` + `go test ./... -short` green.

### Rollout Method
**Incremental**, backward-compatible migration (D5 is additive, defaults existing rows). No flag
needed — the new indexer types are simply unused until a client (SPEC-211) starts sending them.

### Rollback Strategy
The migration's `down` drops the `indexer_type` column; all code paths that read it treat a
missing/default value as `prefixado`, so rollback is safe at any point.

---

## 7. Implementation Phases

### Phase 1 — Domain Layer

#### Tasks
- [x] `internal/portfolio/indexer.go`: `Indexer` closed enum (`prefixado` | `cdi_percentual` |
      `ipca_spread`), `ParseIndexer` (trim+lower, `%w`-wrapped `ErrInvalidIndexer`) — mirrors
      `liquiditytype.go` exactly.
- [x] Add `IndexerType Indexer` to `FixedIncomeHolding` (constructor validates via `ParseIndexer`,
      parse-don't-validate); default `prefixado` when unset.
- [x] `ResolveEffectiveRate(macro map[marketdata.Indicator]marketdata.MacroIndicator) int` method
      (implemented name; same contract as the planned `EffectiveAnnualRateBps`): `prefixado`
      passthrough; `cdi_percentual` → `rate × CDI ÷ 10000` via the existing `money.ApplyBps`
      half-up helper (no bespoke `big.Int` code — reuses the project's one rounding rule);
      `ipca_spread` → `rate + IPCA` — pure, no I/O, doc comment cites FR-1092/D3.

#### Deliverables
- [x] Table-driven unit tests for `ParseIndexer` (`indexer_test.go`) and `ResolveEffectiveRate`
  (`holding_test.go`): all three indexers, the D3 fallback when a reading is absent from the map
  (both nil and empty map), zero-value `Indexer` behaves like Prefixado. Rounding/overflow is
  covered by `money.ApplyBps`'s own test suite (the shared helper this method delegates to).

---

### Phase 2 — Persistence Layer

#### Tasks
- [x] `migrations/0007_fixed_income_indexer.up.sql` — `ALTER TABLE fixed_income_holdings ADD COLUMN
      indexer_type text NOT NULL DEFAULT 'prefixado'` (+ a `CHECK` constraint on the three values,
      matching the project's closed-enum-at-the-DB convention); `.down.sql` drops it.
- [x] `internal/portfolio/postgres/postgres.go`: read/write `indexer_type`, re-validating via
      `ParseIndexer` on read (defense in depth, matches the project's re-validate-on-read convention).

#### Deliverables
- [x] Real-Postgres integration test (`TestRepository_FixedIncomeIndexer_Integration`): a
  `cdi_percentual` row round-trips on create, and a pre-existing row (no `indexer_type` set)
  reads back as `prefixado` — proving BR-1093 backward compatibility.

---

### Phase 3 — Application Layer

#### Tasks
- [x] **Portfolio's own read path** (`GET /holdings/fixed-income`): the service resolves and
      attaches `effective_annual_rate_bps` per holding via its own `MacroReader` (D2) before the
      transport layer maps it into `FixedIncomeResponse`.
- [x] **`internal/dashboard`** / **`internal/projection`**: implemented as a simplification of D2 —
      rather than each service owning its own `MacroReader`, `portfolio.Service.ListHoldings`
      (the single seam dashboard/projection/health/chat all already read holdings through) resolves
      `EffectiveAnnualRateBps` on every FI holding before returning `Holdings`. Dashboard/projection
      consume the already-resolved field via their existing `HoldingsReader` port — no new port, no
      duplicate macro I/O per service, same FR-1093/FR-1094 guarantee. `Compute`'s signature/purity
      is preserved (D4): it still just reads a field off the holding.
- [x] Composition root (`cmd/api` wiring): the existing `marketdata/postgres.MacroRepository`
      instance is passed into `portfolio.NewService`'s single `MacroReader` slot.

#### Deliverables
- [x] Unit tests (hand-written `MacroReader` fake `fakeMacro` in `portfolio/service_test.go`, per
  the project's no-mocking-library convention): Create/List/Update resolve the effective rate.
  `dashboard/compute_test.go` proves a `prefixado`-shaped fixture's `EffectiveAnnualRateBps` equals
  the raw `AnnualRateBps` — the byte-for-byte regression proof required by BR-1093.

---

### Phase 4 — API Layer

#### Tasks
- [x] `GET /market/indicators` handler returning `MarketIndicatorResponse[]` (SELIC/CDI/IPCA, latest
      value + reference date) — behind the standard deny-by-default auth gate; registered in the
      `routeTable`. Owning package: `internal/transport/http` (`market.go`), a thin handler over the
      existing `marketdata.MacroRepository` — matches the "thin handler" precedent.
- [x] Extend `FixedIncomeRequest`/`FixedIncomeResponse` DTOs: accept `indexer_type` on write; return
      it + the computed `effective_annual_rate_bps` on read (never accepted on write — FR-1092).
- [x] Update `api/openapi.yaml`: the two extended schemas + the new path + a `MarketIndicatorResponse`
      schema; `openapi_test.go` drift guard confirmed green.

#### Deliverables
- [x] Handler tests (money round-trip, `indexer_type` validation → `400` on garbage, span carries no
  indicator/holding values per the existing no-PII-on-spans convention).

---

### Phase 5 — Observability

#### Tasks
- [x] Confirm the route-named `otelhttp` span auto-applies to `GET /market/indicators` (no new
      instrumentation code needed — verified via `TestHTTP_MarketIndicatorsSpanRouteNamed`).

#### Deliverables
- [x] `TestHTTP_MarketIndicatorsSpanRouteNamed` asserts the new route's span is present and
  route-named (indicator values are public economic data, not PII, so — unlike holdings — their
  presence on the span is not itself a leak; see the test's own comment).

---

### Phase 6 — Testing

#### Unit Tests
- [x] `Indexer` parsing (`indexer_test.go`); `ResolveEffectiveRate` (`holding_test.go`) — all three
  indexers, D3 fallback (nil and empty map), zero-value defaults to Prefixado.
- [x] `portfolio.Service` tests with a hand-written `MacroReader` fake (`fakeMacro`,
  `portfolio/service_test.go`) covering Create/List/Update.
- [x] `dashboard/compute_test.go` proves a `prefixado`-shaped fixture's output is unchanged
  (`EffectiveAnnualRateBps == AnnualRateBps`); handler-level `indexer_type` round-trip + 400-on-
  garbage + span no-leak tests added in `transport/http/holdings_test.go` (this session, closing
  the Phase 4/6 gap).

#### Integration Tests
- [x] Migration round-trip + backward-compatible default (`TestRepository_FixedIncomeIndexer_Integration`).
- [x] Real-Postgres: `TestService_FixedIncomeIndexer_EffectiveRate_Integration`
  (`portfolio/postgres/service_integration_test.go`) proves Create/List/Update resolve
  `effective_annual_rate_bps` against a real, seeded `marketdata.MacroRepository`;
  `TestService_GetDashboard_FixedIncomeIndexer_Integration` (`dashboard/service_integration_test.go`)
  proves the Dashboard's computed current value reflects it too.
- [x] Regression: `dashboard/compute_test.go`'s `prefixado`-fixture assertion + the repository
  integration subtest "a pre-existing row (no indexer_type set) defaults to Prefixado" together
  prove BR-1093 byte-for-byte backward compatibility.

#### Deliverables
- [x] All green under `task test:integration` against real Postgres (host port 5433) — verified
  this session (`-p 1`, serialized against the shared compose dev DB on 5433).

---

### Phase 7 — Documentation

#### Tasks
- [x] Update `CHANGELOG.md` `[Unreleased]`.
- [x] Flip **SPEC-109 + PLAN-109 → Done**; update both indexes (`docs/02-specs/README.md`,
      `docs/03-plans/README.md`).
- [x] Invoke **lesson-writer** (backend track) for `docs/lessons/SPEC-109-aula.html` (published,
      869 lines, matching SPEC-106/107's structure and depth).
- [x] Note in the closeout that **SPEC-211 is now unblocked** — its `Blocked on` field
      (`docs/02-specs/SPEC-211-portfolio-management-screens.md`) and the frontend specs index
      both updated.

#### Deliverables
- [x] Docs updated, spec closed, lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Silent regression in `prefixado` Dashboard/Projections output | High | Capture before/after fixtures explicitly (Phase 6); this is the single most important regression to prove |
| Three services (`portfolio`, `dashboard`, `projection`) each need their own `MacroReader` wiring | Medium | The pattern is already proven twice (`health`, `insight/engine`) — copy it exactly, don't improvise a fourth shape |
| `GET /market/indicators` placement (which package owns the handler) | Low | Decided in Phase 4 as a thin transport-layer handler over the existing repository — no new domain logic to place |
| Migration `CHECK` constraint too strict if a 4th indexer is added later | Low | Additive migrations are cheap in this project's convention (never edit a committed one) — a future spec adds a new migration to widen the constraint |

---

## 9. Validation Checklist

### Functional Validation
- [x] FR-1091…FR-1095 implemented; SPEC-109 acceptance criteria satisfied.
- [x] BR-1091…BR-1095 respected (integer math, derived-never-stored, backward-compat, graceful
      degradation, ownership/scoping unchanged).

### Technical Validation
- [x] Hexagonal layering intact: no SQL/HTTP in `internal/portfolio` core; `portfolio.MacroReader`
      is a consumer-defined port, not a leak of `marketdata` internals. (Note: dashboard/projection
      don't have their own `MacroReader` as D2 literally described — see Phase 3 note — they
      consume the already-resolved rate through the existing `HoldingsReader` seam instead.)
- [x] `api/openapi.yaml` updated; drift test green (`TestOpenAPI_DocumentsEveryRoute` +
      `TestOpenAPI_NoStaleDocumentedRoutes`).
- [x] Money/rate stays `int64` centavos / integer bps everywhere in the new code.

### Quality Validation
- [x] `task vet` + `task test:short` clean.
- [x] `task test:integration` green against real Postgres.
- [x] Reviewed by **hexagonal-reviewer** (PASS, no blocking issues) + **go-correctness-reviewer**
      (one real bug found and fixed this session: `GetLatestMacroIndicator` errors were swallowed
      indiscriminately in `market.go`'s `getMarketIndicators`, instead of distinguishing
      `marketdata.ErrMacroNotFound` from a real infra failure — now logged via the handler's
      previously-unused `logger` field; `portfolio.Service.latestMacro`'s broader any-error
      fallback was kept, but its doc comment now explicitly justifies the scope beyond PLAN-109
      D3's literal `ErrMacroNotFound` case). A per-indicator-independence unit test was added
      (`TestService_ListHoldings_ResolvesEffectiveRatePerIndicatorIndependently`) to close a
      coverage gap the reviewer flagged.

---

## 10. Definition of Done

- [x] All phases complete; SPEC-109 acceptance criteria satisfied.
- [x] Migration `0007` up/down proven; backward-compatibility regression passes.
- [x] `api/openapi.yaml` updated; drift test green; no other endpoint's contract changed.
- [x] `task vet`, `task test:short`, `go build ./...` clean; `task test:integration` green.
- [x] CHANGELOG updated; SPEC-109 + PLAN-109 flipped to Done; indexes updated.
- [x] PT-BR lesson `docs/lessons/SPEC-109-aula.html` produced.
- [x] Reviewed by the backend review agents (hexagonal-reviewer PASS, go-correctness-reviewer
      CHANGES REQUESTED → fixed → re-verified). Pull Request not yet opened — this is uncommitted
      local work; PR approval is the last remaining step, outside this session's scope.

---

## 11. Deliverables

### Code Deliverables
- `internal/portfolio/indexer.go` + extended `FixedIncomeHolding`; new `MacroReader` ports in
  `dashboard`/`projection`/`portfolio`'s read path; `GET /market/indicators` handler; migration
  `0007`; extended DTOs.

### Documentation Deliverables
- CHANGELOG entry, PT-BR lesson, `api/openapi.yaml` updates, specs/plans index updates.

---

## 12. Post-Implementation Tasks

### Future Improvements
- SPEC-211's fixed-income form consumes this contract (indexer picker + live reference display).
- A possible Dashboard-level "Indicadores de mercado" card (deferred, not decided — see SPEC-211's
  discussion) could reuse `GET /market/indicators` directly.

### Technical Debt
- None anticipated; the migration is additive and the fallback (D3) is a documented, self-healing
  edge case rather than a workaround.
