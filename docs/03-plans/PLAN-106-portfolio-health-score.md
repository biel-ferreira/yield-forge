# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Portfolio Health Score (reproducible computed score + AI narrative) |
| Related Feature | SPEC-106 — a deterministic, market-aware score with a gated narrative |
| Related Spec    | [SPEC-106](../02-specs/SPEC-106-portfolio-health-score.md)   |
| Version         | 0.1.0                                                        |
| Status          | Draft                                                       |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-29                                                   |

---

## 2. Objective

### Goal

Compute a reproducible **0–100 Portfolio Health Score** with a per-factor breakdown — **market-aware**
(macro is an input, deterministic tilt) — and layer an **optional gated LLM "professor" narrative**
that explains it using the live market, without ever changing the number.

### Expected Outcome

`GET /health-score` returns the same score + byte-identical breakdown for the same
`(portfolio, profile, macro)`, plus a contextual narrative when the LLM is available (degrading to
`narrative_available:false` otherwise). The score number is computed, never LLM-generated.

---

## 3. Scope

### Included

- A pure **scoring core** (`internal/health`): the five factor rules (diversification, concentration,
  liquidity, goal-alignment, risk-exposure), each `0–100`, combined by integer half-up weighted mean
  (Σ weights = 10 000); the goal-alignment + risk factors carry a **modest, documented macro tilt**.
- A thin **service** composing the dashboard (SPEC-103), profile (SPEC-101), holdings (SPEC-102),
  and macro (SPEC-006) reads; then the **gated narrative** via the Insighter (SPEC-005, `health_score`
  task), grounded in the computed score/breakdown/macro, degradable.
- HTTP `GET /health-score` + `routeTable`/OpenAPI + `cmd/api` wiring; observability; tests; closeout.

### Excluded (SPEC-106 §scope)

- Any LLM influence on the **score number** or the structured breakdown (computed only).
- **Historical tracking** / persistence (no tables); Insights/Rebalancing/Projections/Chat.
- Changes to the SPEC-005 gates or a new LLM/market adapter.

---

## 4. Dependencies

### Technical Dependencies

- **SPEC-103** `dashboard.Service.GetDashboard` (allocation/sectors/concentration); **SPEC-101**
  `profile.Reader.GetProfile`; **SPEC-102** `portfolio.Service.ListHoldings` (counts + liquidity);
  **SPEC-006** the macro repo `GetLatestMacroIndicator`; **SPEC-005** `insight.Insighter` (+ `Fake`),
  a new `health_score` task.
- `auth.UserID(ctx)`; the `transport/http` router `Deps`/`writeJSON`; `internal/platform/money`
  (a small `WeightedMeanBps`/half-up helper for the score).

### New Dependencies

- **None.** Pure stdlib + the existing stack.

### Blocking Decisions (SPEC-106 §14 — all resolved)

- **D1** layered (computed market-aware score + gated narrative) · **D2** `internal/health` ·
  **D3** weights div 25 / conc 25 / liq 15 / goal 20 / risk 15 (bps) · **D4** profile-not-set →
  renormalize · **D5** read structured sources directly · **D6** empty → `200 score:0`.
- **Hard prerequisites:** SPEC-103/101/102/006/005 — all Done/merged.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/money` | Small `WeightedMeanBps` (half-up) helper for the score (+ tests) |
| `internal/transport/http` | New `health.go` handler + `Deps.HealthScore`; register `GET /health-score`; document in `api/openapi.yaml` |
| `cmd/api` | Build the health service (reusing dashboard/profile/portfolio/macro + the shared Insighter); wire into `Deps` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/health` | `Factor`/`FactorScore`/`HealthScore`, the pure `Compute`, the factor rules, the service + narrative |

---

## 6. Implementation Strategy

### Approach

Deterministic core first, narrative second — the reproducible number is proven before any LLM
enters. Throughlines: **the score is computed, never generated** (and reproducible to the unit, incl.
the macro input); **integer-only** (sub-scores/score in `[0,100]`, weights bps, half-up, weights
reconcile to 10 000); **market-awareness via a documented deterministic rule** (same macro → same
tilt); **guards by construction** for the narrative (text only via the gated Insighter; it never
touches the number). Identity from context. Conventions: closed enums parse-don't-validate (`Factor`),
errors `%w`, consumer-defined interfaces, DTOs separate, doc comments cite SPEC/BR, test files mirror
source, hand fakes + `testify/require`.

### Rollout Method

Incremental, additive, read-only — a new auth endpoint over existing data; no schema change. The
`fake` Insighter keeps the narrative deterministic in dev/CI; the score is deterministic always.

### Rollback Strategy

Remove the endpoint + wiring + the `internal/health` package + the `money` helper. No migration/data.

---

## 7. Implementation Phases

### Phase 1 — Domain & the five factor rules + the weighted score (the heart)

#### Tasks

- [ ] `Factor` closed enum + `FactorScore{Factor, Score, WeightBps, Explanation}` + `HealthScore{Score,
      Factors}`; the default weights (D3).
- [ ] `money.WeightedMeanBps(values []int, weightsBps []int) int` — integer half-up weighted mean,
      Σ weights = 10 000 (+ tests).
- [ ] The five pure factor rules over a structured `Inputs` (dashboard + profile + holdings + macro):
      diversification, concentration, liquidity, goal-alignment (**market-aware**), risk-exposure
      (**market-aware**); each returns a `0–100` sub-score + a templated, reproducible explanation.
- [ ] `Compute(Inputs) HealthScore` — combine via `WeightedMeanBps`; profile-not-set → renormalize (D4);
      empty portfolio → `score:0` + "add holdings" (D6); missing macro → neutral tilt.

#### Deliverables

- Pure, compiling `internal/health` core. Table-driven tests per factor (boundaries: empty,
  single-sector, all-illiquid, profile-unset, macro-present/absent); **reproducibility** (same inputs →
  same score + byte-identical breakdown); the **market tilt** (same macro → same tilt, modest, bounded);
  score ∈ `[0,100]`; weights reconcile; **no float**.

---

### Phase 2 — Inputs & the service (compose the reads)

#### Tasks

- [ ] Consumer-defined ports `DashboardReader`, `ProfileReader`, `HoldingsReader`, `MacroReader`.
- [ ] `health.Service` composing the four reads into `Inputs`, calling `Compute`; profile/macro
      degrade gracefully (renormalize / neutral). No LLM yet.

#### Deliverables

- The service returns a deterministic `HealthScore` over fakes; unit tests for the composition +
  degradation paths.

---

### Phase 3 — The gated AI narrative (guards by construction)

#### Tasks

- [ ] After computing the score, request a narrative via `Insighter.Generate` (`health_score` task)
      with facts = the computed score + factor sub-scores + macro + key portfolio facts; attach the
      gated text + the disclaimer to the result.
- [ ] Degrade gracefully: an Insighter error → `NarrativeAvailable:false`, the score + breakdown
      intact; abort on `ctx.Err()`; the narrative is emitted **only** via the Insighter (marker test).

#### Deliverables

- The service produces a narrative via the gate; tests: only-via-Insighter, degradation →
  unavailable, the score/breakdown unchanged whether or not the narrative succeeds.

---

### Phase 4 — API (transport)

#### Tasks

- [ ] `internal/transport/http/health.go`: `GET /health-score` → DTO `{score, factors:[{name, score,
      weight_bps, explanation}], narrative, narrative_available, disclaimer}`, identity from
      `auth.UserID(ctx)`; service error → 500.
- [ ] `Deps.HealthScore`; register in the `routeTable`; **add the path + schema to `api/openapi.yaml`**
      (score/weights integers; drift test green); wire the service in `cmd/api` (reuse the shared Insighter).

#### Deliverables

- Working endpoint behind auth; handler unit tests (identity, `200` shape, narrative-degraded, `401`);
  OpenAPI drift green.

---

### Phase 5 — Observability

#### Tasks

- [ ] Confirm the route span; add a `health.compute` span (no content); the Insighter records its
      narrative span. **No PII** (no score/holdings/profile/narrative on spans). Optional
      `health_score.requests` counter.

#### Deliverables

- Endpoint traced; a span-no-PII test.

---

### Phase 6 — Testing

#### Unit Tests

- [ ] Each factor rule; the weighted mean + half-up; the market tilt; reproducibility (byte-identical
      breakdown); narrative gated/degraded; empty + profile-unset; handler (identity, shape, `401`).

#### Integration Tests (gated)

- [ ] Real Postgres + the `fake` Insighter: seed holdings + profile + quotes + macro,
      `GET /health-score`, assert the score ∈ `[0,100]`, every factor explained, **two calls return an
      identical score + breakdown** (reproducibility end to end), the narrative carries the disclaimer,
      per-user isolation.

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 7 — Documentation & Lesson

#### Tasks

- [ ] `README` (`/health-score`) + `CHANGELOG`; OpenAPI in lockstep.
- [ ] Flip SPEC-106 + PLAN-106 → **Done**; update indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-106-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| The LLM influences the score number (reproducibility break) | High | The number is computed by `Compute` before any LLM call; the narrative receives the score as a FACT and only explains it; a test asserts the score is identical with the narrative stubbed off. |
| Non-reproducible breakdown (float / map-order / time) | High | Integer-only; templated breakdown in a fixed factor order; no `time.Now()`; byte-identical reproducibility test. |
| Market tilt reads as a market-timing call / confuses trackers | Medium | Keep the tilt modest + documented + bounded; the narrative explains "changed due to market, not your holdings"; §15 leaves tuning open. |
| Narrative bypasses the gates | High | Narrative emitted only via `Insighter.Generate`; gates fail-closed (SPEC-005); marker test. |
| Prompt-injection via holdings in the narrative facts | Low | The score is computed (injection can't move it); the non-advice gate is the narrative backstop; `/security-review` at close. |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1061…FR-1068 implemented; BR-1061…BR-1064 respected; acceptance criteria met.
- [ ] Score reproducible + market-aware; every factor explained; weights reconcile; narrative gated +
      degradable and never changes the number.

### Technical Validation

- [ ] Hexagonal (pure `Compute`; service composes reads + the Insighter at the edge; acyclic); money/
      score integers; identity from context; conventions.

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
      hexagonal-reviewer + go-correctness-reviewer pass; `/security-review` (narrative AI surface).

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] Reproducible market-aware score + byte-identical breakdown against real Postgres; the narrative
      gated + degradable; the number never LLM-touched.
- [ ] CHANGELOG + README updated; OpenAPI in lockstep; SPEC-106 + PLAN-106 → **Done**; indexes +
      `CLAUDE.md` status updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `money.WeightedMeanBps`; `internal/health/**` (domain, factor rules, Compute, service, narrative);
  `internal/transport/http/health.go`; `cmd/api` wiring; `api/openapi.yaml` update.

### Infrastructure Deliverables

- None (no migration; read-only feature).

### Documentation Deliverables

- README endpoint, CHANGELOG entry, `SPEC-106-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch `health_score.requests` (incl. narrative-degraded rate) once the UI consumes it.

### Future Improvements

- Historical tracking (store score + factor snapshot — shows portfolio AND market drift); profile-
  weighted factor weights; tuning the market-tilt aggressiveness; SPEC-108 surfaces the score in chat.

### Technical Debt

- The factor weights + the market-tilt rule are fixed heuristics until tuning/personalisation lands.
