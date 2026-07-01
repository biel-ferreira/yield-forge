# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Projections (deterministic income + net-worth scenarios)     |
| Related Feature | SPEC-107 — the planning layer; pure reproducible projections |
| Related Spec    | [SPEC-107](../02-specs/SPEC-107-projections.md)             |
| Version         | 0.1.0                                                        |
| Status          | Done                                                        |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-30                                                   |

---

## 2. Objective

### Goal

Compute two reproducible, chartable projections over the current portfolio — **passive income**
(monthly/annual, three scenarios) and **net worth** (value over a configurable horizon with a
configurable monthly contribution, three scenarios) — each with its assumptions shown, as a **pure
deterministic computation** (no LLM).

### Expected Outcome

`GET /projections?monthly_contribution_centavos=&horizon_years=` returns the income + net-worth
scenarios + the estimate disclaimer; the same `(holdings, market, contribution, horizon)` always
yields the same figures and series. Money int64 centavos / bps, never a float.

---

## 3. Scope

### Included

- A pure **projection engine** (`internal/projection`): the `Scenario` enum, the income calculation
  (blended FII + FI yield, ±200 bps scenarios), the net-worth **monthly compounding** (reinvested
  income + contribution, yearly snapshots), and `Compute`.
- A thin **service** composing the dashboard (SPEC-103, current value + FII income) + holdings
  (SPEC-102, FI invested + rates) reads into `Inputs`.
- HTTP `GET /projections` (auth) with query-param parsing + `routeTable`/OpenAPI + `cmd/api` wiring;
  observability; tests; closeout.

### Excluded (SPEC-107 §scope)

- Any LLM narrative (deferred; SPEC-108 narrates projections). Capital-appreciation modelling (income
  + contribution only). Persistence; inflation/tax adjustment (nominal, stated). 104/105/106/108.

---

## 4. Dependencies

### Technical Dependencies

- **SPEC-103** `dashboard.Service.GetDashboard` (current value + FII monthly income); **SPEC-102**
  `portfolio.Service.ListHoldings` (fixed-income invested + `AnnualRateBps`).
- `auth.UserID(ctx)`; the `transport/http` router `Deps`/`writeJSON`; `internal/platform/money`
  (`ApplyBps` for half-up monthly compounding, `ShareBps` for the blended yield).

### New Dependencies

- **None.** Pure stdlib + the existing stack. No Insighter (deterministic feature).

### Blocking Decisions (SPEC-107 §14 — all resolved)

- **D1** deterministic, no LLM · **D2** one `GET /projections` · **D3** `internal/projection` ·
  **D4** monthly compounding, yearly points · **D5** base ±200 bps · **D6** income-yield +
  contribution only · **D7** query params (contribution ≥ 0, horizon 1–40; defaults 0/10).
- **Hard prerequisites:** SPEC-103 + SPEC-102 — Done/merged.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/transport/http` | New `projections.go` handler + `Deps.Projections`; register `GET /projections`; document in `api/openapi.yaml` |
| `cmd/api` | Build the projection service (reuse dashboard + portfolio); wire into `Deps` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/projection` | `Scenario`/`ScenarioIncome`/`ScenarioNetWorth`/`Assumptions`/`Projections`, the pure `Compute` + the compounding, the service |

---

## 6. Implementation Strategy

### Approach

Deterministic core first, heavily tested for reproducibility + overflow before the API. Throughlines:
**pure, reproducible computation** (same inputs → same figures + series; no `time.Now`, no float),
**integer money discipline** (int64 centavos / bps, half-up monthly compounding via `money.ApplyBps`,
big.Int-guarded), and **transparent assumptions** (each scenario exposes its bps adjustments). FR-014
holds by construction (a labelled estimate, computed figures, no order). Identity from context; the
only client inputs are the contribution + horizon (parsed, bounded). Conventions: closed enum
`Scenario` parse-don't-validate; errors `%w`; DTOs separate; doc comments cite SPEC/BR; hand fakes +
`testify/require`; test files mirror source.

### Rollout Method

Incremental, additive, read-only — a new auth endpoint over existing data; no schema change, no LLM.

### Rollback Strategy

Remove the endpoint + wiring + the `internal/projection` package. No migration/data to revert.

---

## 7. Implementation Phases

### Phase 1 — Domain & the projection engine (the heart)

#### Tasks

- [ ] `Scenario` closed enum (`pessimistic|base|optimistic`) + `ParseScenario`; the `Assumptions`,
      `ScenarioIncome`, `NetWorthPoint`, `ScenarioNetWorth`, `Projections` types; the ±200 bps spread
      constants (D5).
- [ ] `Inputs` (flat: `CurrentValueCentavos`, `FIIAnnualIncomeCentavos`, `FIAnnualIncomeCentavos`,
      `MonthlyContributionCentavos`, `HorizonYears`).
- [ ] **Income**: base annual income = FII + FI; base blended yield = `ShareBps(income, value)`;
      per scenario apply the yield ±200 bps → `ApplyBps(value, scenarioYield)`; monthly = annual/12
      (half-up); expose assumptions.
- [ ] **Net worth**: per scenario, compound monthly — `value = value + ApplyBps(value, yieldBps/12)
      + contribution` over `12×horizon` months; snapshot at each year → `[]NetWorthPoint`; expose assumptions.
- [ ] `Compute(Inputs) Projections`.

#### Deliverables

- Pure, compiling `internal/projection`. Table-driven tests: income base reconciles + haircut/uplift;
  net-worth compounding against hand-computed expected series; **reproducibility** (same inputs →
  identical output); empty portfolio (income 0, net worth = future value of contributions);
  **overflow safety** at 40 years; no float.

---

### Phase 2 — Inputs & the service (compose the reads)

#### Tasks

- [ ] Consumer-defined ports `DashboardReader`, `HoldingsReader`.
- [ ] `projection.Service.Project(ctx, userID, contribution, horizon)` — read the dashboard (current
      value + FII annual income = `MonthlyIncome×12`) + holdings (Σ FI `invested×AnnualRateBps`),
      assemble `Inputs`, run `Compute`. Dashboard/holdings errors surface.

#### Deliverables

- The service returns a deterministic `Projections` over fakes; unit tests for the FI-income
  derivation + the composition.

---

### Phase 3 — API (transport)

#### Tasks

- [ ] `internal/transport/http/projections.go`: `GET /projections` → parse `monthly_contribution_centavos`
      (≥ 0) + `horizon_years` (1–40) from the query (integers, **never float**; defaults 0/10; bad →
      `400`), identity from `auth.UserID(ctx)`; response DTO (income + net_worth scenarios + assumptions
      + disclaimer); service error → 500.
- [ ] `Deps.Projections`; register in the `routeTable`; **add the path + schema to `api/openapi.yaml`**
      (money as integers; drift test green); wire the service in `cmd/api`.

#### Deliverables

- Working endpoint behind auth; handler unit tests (identity, param parse/defaults, `400` on bad
  params, `200` shape, `401`); OpenAPI drift green.

---

### Phase 4 — Observability

#### Tasks

- [ ] Confirm the `GET /projections` route span; add a `projection.compute` span (no content).
      **No PII** (no figures/holdings/series on spans). Optional `projections.requests` counter.

#### Deliverables

- Endpoint traced; a span-no-PII test.

---

### Phase 5 — Testing

#### Unit Tests

- [ ] Income scenarios; net-worth compounding + monotonicity; scenario parsing; reproducibility;
      overflow; handler (identity, params, shapes, `401`).

#### Integration Tests (gated)

- [ ] Real Postgres: seed holdings + quotes, `GET /projections`, assert the base income reconciles
      with the seeded data, a positive scenario's net-worth series is monotonic, **two calls return an
      identical projection** (reproducibility end to end), per-user isolation.

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 6 — Documentation & Lesson

#### Tasks

- [ ] `README` (`/projections`) + `CHANGELOG`; OpenAPI in lockstep.
- [ ] Flip SPEC-107 + PLAN-107 → **Done**; update indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-107-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Non-reproducible projection (float / time / rounding drift) | High | Integer-only; `money.ApplyBps` half-up; no `time.Now` (year offsets, not calendar); byte-identical reproducibility test. |
| Integer overflow over a long horizon | Medium | `ApplyBps` uses big.Int; net worth in int64 centavos is safe to ~R$9e14; horizon bounded to 40y; explicit overflow test. |
| Projection mistaken for a guarantee / advice | Medium | Labelled estimate + shown assumptions + non-advice disclaimer (FR-014); no LLM, so no order signature possible. |
| Compounding math wrong (off-by-one on months/years) | Medium | Hand-computed expected series in tests; snapshot exactly at year boundaries; monotonicity check. |
| Base income mis-reconciles with the dashboard | Low | Reuse `dashboard.Summary.MonthlyIncomeCentavos` for FII; FI from holdings; a reconciliation test. |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1071…FR-1077 implemented; BR-1071…BR-1074 respected; acceptance criteria met.
- [ ] Deterministic + reproducible; income reconciles; net-worth compounds correctly; assumptions +
      disclaimer shown; integer-only.

### Technical Validation

- [ ] Hexagonal (pure `Compute`; service composes reads; acyclic; no SQL/HTTP/LLM in core); money int64
      centavos/bps incl. the wire; identity from context; conventions.

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
      hexagonal-reviewer + go-correctness-reviewer pass.

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] Reproducible projections against real Postgres (two calls identical); income reconciles;
      net-worth compounds; integer-only; assumptions + disclaimer present.
- [ ] CHANGELOG + README updated; OpenAPI in lockstep; SPEC-107 + PLAN-107 → **Done**; indexes +
      `CLAUDE.md` status updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `internal/projection/**` (domain, Compute, compounding, service); `internal/transport/http/projections.go`;
  `cmd/api` wiring; `api/openapi.yaml` update.

### Infrastructure Deliverables

- None (no migration; read-only feature).

### Documentation Deliverables

- README endpoint, CHANGELOG entry, `SPEC-107-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch `projections.requests` once the UI consumes it.

### Future Improvements

- An optional LLM narrative of the trajectory (SPEC-108 reads these projections); capital-appreciation
  modelling; a real-terms (IPCA-adjusted) toggle; a saved monthly contribution on the profile.

### Technical Debt

- The ±200 bps scenario spread + the income-only return basis are fixed heuristics until tuning lands.
