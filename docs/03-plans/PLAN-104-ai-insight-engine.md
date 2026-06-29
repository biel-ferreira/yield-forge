# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | AI Insight Engine (Fact Builder + explainable insights)      |
| Related Feature | SPEC-104 — the capstone AI feature; the published Fact Builder seam |
| Related Spec    | [SPEC-104](../02-specs/SPEC-104-ai-insight-engine.md)       |
| Version         | 0.1.0                                                        |
| Status          | Approved (decisions D1–D6 resolved)                          |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-29                                                   |

---

## 2. Objective

### Goal

Generate explainable Portfolio / Allocation / Market-Context insights (FR-008/009/010) by
building deterministic facts (the **published Fact Builder**) and emitting them **only** through
the SPEC-005 `Insighter` (gates included), per-user and behind auth.

### Expected Outcome

`GET /insights` returns the caller's gated, category-tagged insights + the non-advice
disclaimer, grounded in computed facts. The Fact Builder is an exported `BuildFacts(ctx, userID)`
seam the Conversational Copilot (SPEC-108) and SPEC-105/106 reuse without re-implementation.

---

## 3. Scope

### Included

- `internal/insight` (D1): the **Fact Builder** (`BuildFacts`, published seam), the engine
  **service** (build facts → call Insighter per category → aggregate), the `Category`↔`insight.Task`
  values, and the consumer input ports (`DashboardReader`, `ProfileReader`, `MacroReader`).
- HTTP `GET /insights` (auth-protected) + `routeTable`/OpenAPI registration + `cmd/api` wiring.
- Graceful degradation + empty state; observability (reuses the Insighter AI telemetry); tests.

### Excluded (SPEC-104 §2)

- Rebalancing (SPEC-105), Health Score (SPEC-106), Projections (SPEC-107), Chat (SPEC-108).
- **Insight history / persistence (FR-022)** — deferred (D4); no new tables/migration.
- Any new LLM adapter or change to the gates (SPEC-005), any new market-data source (SPEC-006).

---

## 4. Dependencies

### Technical Dependencies

- **SPEC-005** — `insight.Insighter` (gates + cache + degradation + AI telemetry), `insight.Facts`,
  `insight.InsightRequest`/`InsightResult`, `insight.Fake`, and the `Task` type (new category values).
- **SPEC-103** — `dashboard.Service.GetDashboard` (the allocation/sector/concentration facts, D3).
- **SPEC-101** — `profile.Reader.GetProfile`. **SPEC-006** — the macro repo `GetLatestMacroIndicator`.
- `auth.UserID(ctx)`, the `transport/http` router `Deps`/`writeJSON`, the `money` units (centavos/bps).

### New Dependencies

- **None.** Pure stdlib + the existing stack.

### Blocking Decisions (SPEC-104 §14 — all resolved)

- **D1** engine in `internal/insight` · **D2** one fact set, N category calls · **D3** reuse the
  dashboard · **D4** defer insight-history (FR-022; conversation memory is SPEC-108) · **D5**
  degraded = `200` `available:false` · **D6** Fact Builder is a published `BuildFacts` seam.
- **Hard prerequisite:** SPEC-103 (dashboard) must be merged — the Fact Builder reuses
  `GetDashboard`. (SPEC-005/101/006 are already Done.)

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/insight` | Gains the Fact Builder + engine service + category tasks (the core/port stay pure; new file(s), no change to the gates) |
| `internal/insight/factory` | Expose the composed `Insighter` to wire into the engine (likely already returned by `factory.New`) |
| `internal/transport/http` | New `insights.go` handler + `Deps.Insights`; register `GET /insights` in `routeTable`; document in `api/openapi.yaml` |
| `cmd/api` | Build the Insighter (factory), the Fact Builder (dashboard+profile+macro readers), the engine; wire into `Deps` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/insight` Fact Builder + engine | Deterministic facts + per-category Insighter orchestration |

---

## 6. Implementation Strategy

### Approach

Bottom-up, with **"facts computed, not generated" and "guards by construction" as the two
throughlines**. The Fact Builder is a deterministic function of the read seams (no `time.Now()`
beyond the Clock the dashboard already injects; money `int64` centavos / integer bps, never
float) — and it is **exported** as `BuildFacts(ctx, userID)` so SPEC-108/105/106 reuse it. The
engine emits user-facing AI text **only** through the `Insighter` (the gates are unavoidable —
the engine never constructs ungated strings). Identity is `auth.UserID(ctx)`. Conventions:
errors `%w` + sentinels; `ctx`-first; closed enums (`Category`) parse-don't-validate; DTOs
separate from domain; no package stutter; doc comments cite SPEC/BR; test files mirror source.

### Rollout Method

Incremental, additive, read-only — a new auth-protected endpoint over existing data; no schema
change. The Insighter default (`fake`) keeps dev/CI deterministic and offline.

### Rollback Strategy

Remove the endpoint + wiring. No migration/data to revert; the Fact Builder/engine are additive.

---

## 7. Implementation Phases

### Phase 1 — Domain, Tasks & Input Ports

#### Tasks

- [ ] `Category` closed enum (`portfolio|allocation|market_context`) ↔ `insight.Task` values; the
      engine result type `Insights{ Items []insight.Insight; Disclaimer string; Available bool }`.
- [ ] Consumer input ports `DashboardReader`, `ProfileReader`, `MacroReader`; the published
      `FactBuilder` interface (`BuildFacts(ctx, userID) (insight.Facts, error)`); sentinels.

#### Deliverables

- Compiling pure additions to `internal/insight` (no SQL/HTTP); enum + type unit tests.

---

### Phase 2 — The Fact Builder (the heart)

#### Tasks

- [ ] `BuildFacts(ctx, userID)`: read the dashboard (current value, allocation bps, sector bps,
      largest-holding concentration, stale tickers), the profile (risk/objectives/horizon), and the
      latest macro (SELIC/CDI/IPCA) → assemble a deterministic `insight.Facts` (money centavos /
      rates bps). Empty portfolio → minimal/empty-state facts; profile-not-set → omit profile fields;
      missing macro → omit that fact.

#### Deliverables

- Table-driven tests: known dashboard/profile/macro → **expected deterministic facts**; empty,
  profile-not-set, missing-macro; **no float** in any fact value.

---

### Phase 3 — The Engine (gates by construction)

#### Tasks

- [ ] `Service` (or `Engine`): `Insights(ctx, userID) (Insights, error)` — short-circuit the empty
      portfolio (no LLM call); otherwise `BuildFacts` once, then call `Insighter.Generate` per
      category (tagging each returned insight with its category), aggregate the gated results +
      disclaimer; map `insight.ErrInsightsUnavailable` per category → partial/`Available:false`.
- [ ] Hand-written fakes (the deterministic `insight.Fake` + fake readers) for unit tests.

#### Deliverables

- Engine that emits AI text **only** via the Insighter; unit tests (aggregation, empty state,
  degradation → unavailable, partial success, and an assertion that no AI text bypasses the port).

---

### Phase 4 — API (transport)

#### Tasks

- [ ] `internal/transport/http/insights.go`: `GET /insights` → response DTO (insights tagged by
      category + disclaimer + `available`), identity from `auth.UserID(ctx)`; empty/degraded →
      `200` shapes (D5); service error → 500.
- [ ] Add `Deps.Insights`; register in the `routeTable`; **add the path + schema to
      `api/openapi.yaml`** (drift test green); wire the engine in `cmd/api`.

#### Deliverables

- Working endpoint behind auth; handler unit tests (identity-from-context, empty `200`, degraded
  `available:false`, 401); OpenAPI drift test green.

---

### Phase 5 — Observability

#### Tasks

- [ ] Confirm the `GET /insights` route span; add a `insight.facts` span for fact-building (**no
      PII**); the Insighter records its `insight.generate` spans per category (SPEC-005, no content).
      Optional `insights.requests` counter by outcome.

#### Deliverables

- Endpoint traced; a span-no-PII test (no facts/figures/generated text on spans).

---

### Phase 6 — Testing

#### Unit Tests

- [ ] Fact Builder (determinism/empty/profile-not-set/missing-macro); engine (aggregate, empty,
      degrade, partial, only-via-Insighter); handler (identity, shapes, 401).

#### Integration Tests (gated)

- [ ] Real Postgres + the **`fake` Insighter**: seed holdings + quotes + profile + macro, call
      `GET /insights`, assert **every insight carries an explanation** and the **disclaimer is
      present** (the gates hold end-to-end), per-user isolation.

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 7 — Documentation & Lesson

#### Tasks

- [ ] `README` (AI insights — add the `/insights` endpoint) + `CHANGELOG` entry; OpenAPI in lockstep.
- [ ] Flip SPEC-104 + PLAN-104 to **Done**; update indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-104-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| AI output bypasses the gates (a path that builds user-facing text without the Insighter) | High | The engine emits AI text **only** via `Insighter.Generate`; an explicit test asserts no other source; the gates live in SPEC-005, fail-closed. |
| Non-deterministic / float facts (breaks "facts computed, not generated") | High | Fact Builder is a pure function of the read seams; money int64 centavos / bps; determinism + no-float tests. |
| Prompt-injection via holdings/profile values in the facts | Medium | Facts passed as structured values (not free narrative); the non-advice gate is the backstop; `/security-review` at close. |
| 3 LLM calls per request feel slow / hit free-tier limits | Medium | The SPEC-005 cache keys by (user, task, facts) so repeats are free; graceful degradation; the `fake` provider for dev/CI. |
| SPEC-103 not merged when 104 builds (Fact Builder reuses GetDashboard) | Medium | Hard prerequisite flagged; confirm 103 is merged before `/spec-implement 104`. |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1041…FR-1050 implemented; BR-1041…BR-1047 respected; acceptance criteria met.
- [ ] Every returned insight carries an explanation + the disclaimer (gates hold); facts deterministic.

### Technical Validation

- [ ] Hexagonal (engine composes read seams + Insighter port, acyclic, pure core); `FactBuilder`
      exported as the reusable seam; money int64 centavos/bps; identity from context; conventions.

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
      hexagonal-reviewer + go-correctness-reviewer pass; `/security-review` suggested (AI safety).

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] Gates hold end-to-end (explanation + disclaimer on every insight) against real Postgres +
      the fake Insighter; Fact Builder deterministic + exported as `BuildFacts`.
- [ ] CHANGELOG + README updated; OpenAPI in lockstep; SPEC-104 + PLAN-104 → **Done**; indexes +
      `CLAUDE.md` status updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `internal/insight` Fact Builder + engine + category tasks, `internal/transport/http/insights.go`,
  `cmd/api` wiring, `api/openapi.yaml` update.

### Infrastructure Deliverables

- None (no migration; read-only feature).

### Documentation Deliverables

- README endpoint, CHANGELOG entry, `SPEC-104-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch the Insighter generation outcomes + (optional) `insights.requests` once the UI consumes it.

### Future Improvements

- SPEC-108 (chat) reuses `BuildFacts`; SPEC-105/106 reuse/extend it; FR-022 insight-history as a
  focused follow-up; an explicit target-allocation source for allocation insights (with SPEC-107).

### Technical Debt

- Allocation insights reason qualitatively vs the risk profile until explicit targets exist (D-open).
