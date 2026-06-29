# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | AI Rebalancing Assistant (computed split + grounded candidates) |
| Related Feature | SPEC-105 — second consumer of the published Fact Builder seam |
| Related Spec    | [SPEC-105](../02-specs/SPEC-105-ai-rebalancing-assistant.md) |
| Version         | 0.1.0                                                        |
| Status          | Draft                                                       |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-29                                                   |

---

## 2. Objective

### Goal

Turn a **contribution amount** into explainable guidance — suggested **areas** (each with a
**deterministically computed share** of the contribution), optional **grounded named candidates**,
and the non-advice disclaimer — by reusing the SPEC-104 Fact Builder and emitting **only** through
the gated `Insighter` (SPEC-005).

### Expected Outcome

`POST /rebalancing` returns per-area suggestions with a computed `suggested_share_bps` (Σ = 10 000),
named FII candidates grounded in the live universe (hallucinated tickers dropped), and the
disclaimer — every item explained, no transaction order. The **% is a computed fact** the LLM only
explains; live market data tilts the **narrative**, not the number (D7-A).

---

## 3. Scope

### Included

- A reusable **`money.AllocateBps`** primitive (split 10 000 bps across weights, half-up,
  remainder absorbed, Σ = 10 000 exactly) + a **`rebalancing.ContributionSplit`** allocator (class
  weights from current allocation + the profile-implied direction).
- A new feature package **`internal/rebalancing`**: the `Contribution` value object, the
  `rebalancing` `insight.Task`, the engine (reuse `BuildFacts` → augment → call Insighter → ground
  candidates → join shares), and the result types.
- A new market-data read **`ListFIIUniverse`** (query over the existing `fii_quotes`) + the
  `UniverseReader` port.
- HTTP `POST /rebalancing` (auth) with the `include_asset_shares` flag + `routeTable`/OpenAPI +
  `cmd/api` wiring; observability; tests; closeout.

### Excluded (SPEC-105 §scope)

- Numeric per-class **target** allocations (D4 — qualitative direction only).
- A **market-tilted split number** (D7 — narrative only in the MVP; documented fast-follow).
- Named-product universe beyond FIIs (FI candidates stay type-level); stocks/ETFs.
- Persisting rebalancing history; any change to the SPEC-005 gates; new LLM/market adapters.

---

## 4. Dependencies

### Technical Dependencies

- **SPEC-104** — the published `engine.FactBuilder` / `BuildFacts(ctx, userID)` seam (portfolio facts).
- **SPEC-005** — `insight.Insighter` (gated chain), `insight.Facts/Insight/InsightRequest`, `Fake`,
  a new `rebalancing` `Task`.
- **SPEC-006** — the `fii_quotes` table + `marketdata.FIIQuote`; the macro already in `BuildFacts`.
- **SPEC-103** — `money.ShareBps` discipline + `dashboard.ClassSlice`/`AssetClass` (the allocation shape).
- **SPEC-101** — `profile.Profile` (risk → the split direction). `auth.UserID(ctx)`; the
  `transport/http` router `Deps`/`writeJSON`; `internal/platform/money`.

### New Dependencies

- **None.** Pure stdlib + the existing stack.

### Blocking Decisions (SPEC-105 §14 — all resolved)

- **D1** engine in `internal/rebalancing` · **D2** named FII candidates in MVP (+ universe read) ·
  **D3** areas/candidates as gated `insight.Insight` tagged by category · **D4** qualitative targets ·
  **D5** degraded = `200 available:false` · **D6** area-level % default, per-asset % opt-in
  (`include_asset_shares`) · **D7** market tilts the **narrative**, not the split number.
- **Hard prerequisite:** SPEC-104 merged (the engine reuses `BuildFacts`). ✅ Done.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/money` | New `AllocateBps` primitive (+ tests); reuses the half-up rule (ratio.go) |
| `internal/marketdata` + `…/postgres` | New `ListFIIUniverse` read (query over `fii_quotes`; no schema change) |
| `internal/transport/http` | New `rebalancing.go` handler + `Deps.Rebalancing`; register `POST /rebalancing`; document in `api/openapi.yaml` |
| `cmd/api` | Build the universe reader, the allocator wiring, the rebalancing engine; wire into `Deps` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/rebalancing` | `Contribution` VO, `ContributionSplit` allocator, the engine, result types |
| `money.AllocateBps` | Reusable Σ-10 000 weighted split (half-up, remainder-absorbing) |

---

## 6. Implementation Strategy

### Approach

Bottom-up, with the same two throughlines as SPEC-104 plus a third: **facts computed not generated**
(here the % split is a deterministic fact, the names are grounded), **guards by construction** (text
only through the `Insighter`), and **money is integer + reconciles** (the split sums to exactly
10 000 bps, half-up, like the dashboard). The allocator is pure and lands first so the headline
number is proven before any LLM enters. Identity is `auth.UserID(ctx)`; the contribution is the only
client input (parsed `> 0`). Conventions throughout: errors `%w` + sentinels; `ctx`-first; closed
enums parse-don't-validate; DTOs separate from domain; no package stutter; doc comments cite SPEC/BR;
test files mirror source; hand-written fakes + `testify/require`.

### Rollout Method

Incremental, additive: a new auth-protected endpoint + one new read query (no schema change). The
`fake` Insighter keeps dev/CI deterministic and offline.

### Rollback Strategy

Remove the endpoint + wiring + the `ListFIIUniverse` query. The `money.AllocateBps` primitive and
the `rebalancing` package are additive; no migration/data to revert.

---

## 7. Implementation Phases

### Phase 1 — Domain & the deterministic allocator (the heart)

#### Tasks

- [ ] `money.AllocateBps(weights []int64) []int` — split 10 000 bps across non-negative weights,
      **half-up**, largest-remainder absorption so the result **sums to exactly 10 000**; zero total
      weight → empty/zero. Cite SPEC-105 BR-1056.
- [ ] `rebalancing.Contribution` value object (`int64` centavos, `Parse`/constructor validates `> 0`,
      sentinel `ErrInvalidContribution`).
- [ ] `rebalancing.ContributionSplit` — pure `Split(current []dashboard.ClassSlice, risk
      profile.RiskProfile, contribution Contribution) []AreaShare`: derive class weights from the
      current allocation + a small documented **profile-direction** rule (conservative → tilt Fixed
      Income; aggressive → tilt FII), call `AllocateBps`, attach `suggested_amount_centavos`.
- [ ] `rebalancing` result types (`Area`, `Candidate`, `Rebalancing`) + the `rebalancing` `insight.Task`.

#### Deliverables

- Pure, compiling `internal/rebalancing` + `money.AllocateBps`. Table-driven tests: **Σ = 10 000**
  exactly across tricky weights (rounding), determinism, empty-portfolio follows direction, no float.

---

### Phase 2 — Persistence: the FII universe read

#### Tasks

- [ ] `marketdata.FIIQuoteRepository.ListFIIUniverse(ctx) ([]marketdata.FIIQuote, error)` — `SELECT`
      over `fii_quotes` (same columns as `GetFIIQuoteByTicker`, no `WHERE`), ordered deterministically.
- [ ] The consumer-defined `rebalancing.UniverseReader` port (satisfied by the repo at the edge).

#### Deliverables

- The query + a gated integration test (real PG: seed quotes → list returns them). No schema change.

---

### Phase 3 — Rebalancing facts (augment the Fact Builder)

#### Tasks

- [ ] `rebalancing` fact assembly: reuse `engine.BuildFacts(ctx, userID)`, then add
      `contribution_centavos`, the **FII universe** (ticker/sector/yield, from `UniverseReader`), and
      the **computed split** (`AreaShare` → bps + centavos). Integers only; deterministic.

#### Deliverables

- Unit tests: facts include contribution + universe + computed split; reuses `BuildFacts` (fake seam);
  deterministic; **no float**; the split inside the facts reconciles to 10 000.

---

### Phase 4 — The Engine (orchestration, grounding guard, gates by construction)

#### Tasks

- [ ] `rebalancing.Service.Rebalance(ctx, userID, contribution, opts)` — build facts, call the
      Insighter once for the `rebalancing` task, **tag** items as `area`/`candidate`, **join** the
      computed `suggested_share_bps` onto areas, and apply the **grounding guard** (drop any
      `candidate` whose ticker ∉ the universe). Empty portfolio → still guide (no short-circuit).
- [ ] Degradation → `Available:false` (consistent with SPEC-104); abort on `ctx.Err()`; the
      per-asset share is attached only when `opts.IncludeAssetShares` (D6).

#### Deliverables

- Engine that emits AI text **only** via the Insighter (marker test); unit tests: areas carry the
  computed share; **grounding guard drops unknown tickers**; empty-portfolio guides; degrade →
  unavailable; partial; per-asset share gated behind the flag.

---

### Phase 5 — API (transport)

#### Tasks

- [ ] `internal/transport/http/rebalancing.go`: `POST /rebalancing` → parse `{contribution_centavos,
      include_asset_shares?}` (`json.Number`/int, **never float**; `> 0` else `400`), identity from
      `auth.UserID(ctx)`; response DTO (areas + `suggested_share_bps` + candidates inside + disclaimer
      + `available`); degraded → `200`; service error → `500`.
- [ ] `Deps.Rebalancing`; register in the `routeTable`; **add the path + schema to `api/openapi.yaml`**
      (money/shares as integers; drift test green); wire the engine in `cmd/api`.

#### Deliverables

- Working endpoint behind auth; handler unit tests (identity, `400` bad amount, `401`, degraded `200`,
  the flag toggles per-asset shares); OpenAPI drift green.

---

### Phase 6 — Observability

#### Tasks

- [ ] Confirm the `POST /rebalancing` route span; reuse the `insight.facts` span; the Insighter
      records its generation span. **No PII** (no contribution amount, facts, or generated text).
      Optional `rebalancing.requests` counter by outcome.

#### Deliverables

- Endpoint traced; a span-no-PII test (no amount/figures/generated text on spans).

---

### Phase 7 — Testing

#### Unit Tests

- [ ] Allocator (Σ = 10 000, determinism, direction, no float); contribution parse; fact augmentation;
      grounding guard; engine (only-via-Insighter, empty guides, degrade, flag); handler (identity,
      `400`/`401`, degraded `200`).

#### Integration Tests (gated)

- [ ] Real Postgres + the **`fake` Insighter**: seed holdings + a quote **universe** + profile,
      `POST /rebalancing` → assert **every area/candidate carries an explanation**, the **disclaimer
      is present** (gates hold end to end), candidates are **grounded** in the seeded universe, areas
      carry shares summing to 10 000, and per-user isolation holds.

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 8 — Documentation & Lesson

#### Tasks

- [ ] `README` (`/rebalancing` endpoint) + `CHANGELOG` entry; OpenAPI in lockstep.
- [ ] Flip SPEC-105 + PLAN-105 → **Done**; update indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-105-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| The % reads as advice / a transaction order | High | Computed **area-level** split = a consideration, outside the FR-014 order signature; per-asset % is opt-in + framed as a consideration; the gate is the backstop; `/security-review` at close. |
| LLM invents a ticker (hallucinated candidate) | High | Deterministic **grounding guard** drops any candidate ∉ the live universe (BR-1053); test-enforced. |
| AI text bypasses the gates | High | Engine emits AI text **only** via `Insighter.Generate`; marker test; gates fail-closed (SPEC-005). |
| Split doesn't reconcile (rounding drift) | Medium | `money.AllocateBps` half-up + largest-remainder → Σ = 10 000 exactly; reconciliation test. |
| LLM alters/echoes the numbers instead of explaining | Medium | Split computed **before** the call, passed as a fact; the response uses the computed values, not parsed-from-text numbers. |
| Empty/large universe perf or prompt bloat | Low | `ListFIIUniverse` is a small table; cap/curate the universe facts to the relevant sectors if needed; cache via the Insighter key. |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1051…FR-1059 implemented; BR-1051…BR-1056 respected; acceptance criteria met.
- [ ] Every area/candidate explained + disclaimer present (gates hold); candidates grounded; split
      reconciles to 10 000; per-asset share only when opted in.

### Technical Validation

- [ ] Hexagonal (engine composes the `BuildFacts` seam + `UniverseReader` + `Insighter`, acyclic, pure
      core); money int64 centavos / bps incl. the wire; identity from context; conventions.

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
      hexagonal-reviewer + go-correctness-reviewer pass; `/security-review` (AI-output safety).

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] Gates hold end-to-end (explanation + disclaimer on every output) against real Postgres + the
      fake Insighter; candidates grounded; the split deterministic + reconciles to 10 000.
- [ ] CHANGELOG + README updated; OpenAPI in lockstep; SPEC-105 + PLAN-105 → **Done**; indexes +
      `CLAUDE.md` status updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `money.AllocateBps`; `internal/rebalancing/**` (VO, allocator, engine, result types);
  `marketdata.ListFIIUniverse`; `internal/transport/http/rebalancing.go`; `cmd/api` wiring;
  `api/openapi.yaml` update.

### Infrastructure Deliverables

- None (no migration; one additive read query).

### Documentation Deliverables

- README endpoint, CHANGELOG entry, `SPEC-105-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch the rebalancing outcomes + (optional) `rebalancing.requests` once the UI consumes it.

### Future Improvements

- The **market-tilted split number** (D7 fast-follow): a documented deterministic rule over rate
  bands. Named **Tesouro Direto** universe for FI candidates. Explicit numeric per-class targets
  (with SPEC-107). SPEC-108 routes the "tenho R$X" turn here and sets `include_asset_shares`.

### Technical Debt

- The profile-direction rule is a small heuristic until explicit target allocations exist (D4/D7).
