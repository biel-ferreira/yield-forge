# PLAN-212 — Dashboard Screen (Painel)

## 1. Document Information

| Field           | Value                                   |
| --------------- | --------------------------------------- |
| Plan Name       | Dashboard Screen (Painel)               |
| Related Feature | Dashboard Screen (Painel)               |
| Related Spec    | [SPEC-212](../02-specs/SPEC-212-dashboard-screen.md) (Done) |
| Version         | 0.1.0                                    |
| Status          | Done                                      |
| Author          | Gabigol                                  |
| Last Updated    | 2026-07-14                               |

> **Phase-order note.** Frontend spec — the template's backend phase order is mapped to its
> frontend analogue (data → components → screen → tests → docs), the same mapping PLAN-210 used.
> This is a **smaller** plan than PLAN-211: a single read-only `GET /dashboard` fetch, no forms,
> no new UI primitive (reuses `AllocationBar`, `Card`, `Badge`, `EmptyState` — all already
> built), so 5 phases rather than 7.

---

## 2. Objective

### Goal

Turn the `/dashboard` stub into the real **Painel** screen (SPEC-212): the hero patrimony card,
key-metrics row, asset-class allocation, and FII sector exposure, all read from the live
SPEC-103 backend — with honest loading/error/empty/stale-data states.

### Expected Outcome

A user with holdings sees their current patrimony, growth vs. cost basis, monthly passive
income, and both allocation breakdowns at a glance, every figure read verbatim from the backend
(never recomputed client-side). A user with no holdings sees a clear empty state pointing to
Carteira. No new endpoint, **no `api/openapi.yaml` change**.

---

## 3. Scope

### Included

- **`lib/dashboard/dashboard.ts`** — `useDashboard()` data hook over `GET /dashboard`.
- **`lib/dashboard/labels.ts`** — the FII sector ↔ pt-BR label map, built defensively (D6, see
  Blocking Decisions — the wire's `sector` field is plain `string`, not a closed enum).
- Two new presentational components grouping the screen's sections (hero + metrics + stale
  notice; asset-class + FII-sector allocation) — see Architecture Impact for the exact split.
- The **Painel screen** (`app/(app)/dashboard/page.tsx`) composing them, with loading/error/
  empty-portfolio states matching SPEC-210/211's established patterns.
- Tests (Vitest/RTL + integration + a gated Playwright E2E) and the SDD closeout.

### Excluded

- **Health Score and AI Insights** — SPEC-213's territory, not touched here (SPEC-212 §2 Scope
  is explicit about this; the current `/dashboard` stub's placeholder copy incorrectly implies
  otherwise — this plan's Phase 3 replaces that copy along with the rest of the stub).
- Any backend change — SPEC-103 is Done; this consumes it only.
- Wiring the shell's global "+ Adicionar ativo" button (resolved: stays a no-op here, SPEC-212
  Open Question #1 / this plan's D3).
- A donut chart or any new chart primitive (resolved: reuse `AllocationBar`, SPEC-212 Open
  Question #2 / this plan's D4).
- ~~Cross-screen cache invalidation from Carteira's mutations~~ — **originally deferred, then
  implemented in Phase 5** once review proved the deferral assumption wrong (D5).
- Any historical/time-series balance view — the backend computes only the current snapshot.

---

## 4. Dependencies

### Technical Dependencies

- **SPEC-103** backend (`GET /dashboard`, the `DashboardResponse` shape) — Done and running.
- **SPEC-200** foundation: the typed client, TanStack Query, the `(app)` shell + `RequireAuth`,
  the Aurora design system, `components/ui/{card,badge,empty-state}.tsx`,
  `components/allocation-bar.tsx`, `lib/money.ts`'s `formatCentavos`/`formatShareBps`.
- **SPEC-210/211** precedent: the `lib/<feature>/{labels,hooks}.ts` module shape, the
  loading/error/empty state patterns, and the 404→`null`-style degradation posture (here: a
  zeroed `200` → the empty state, not an error).

### External Dependencies

None new — this plan introduces no new component, only new data-shaping logic over existing
primitives.

### Blocking Decisions

| # | Decision | Resolution (this plan) |
|---|----------|------------------------|
| D1 | Growth-figure duplication (hero badge + its own metric card, FR-2121/FR-2122) | Both read the **same** `summary.growth_centavos`/`growth_bps` fields through **one** shared formatting call — never two independent reads/branches that could drift apart. |
| D2 | Component granularity | **Two** new presentational files, not five: `summary-hero.tsx` (hero + metric row + stale-ticker notice — everything derived from `summary`/`stale_tickers`) and `allocation-sections.tsx` (asset-class + FII-sector breakdowns — both `AllocationBar`-based, no per-item interactivity, unlike SPEC-211's CRUD verticals which justified more files). Matches "don't over-abstract" — this screen has no forms/modals to separate out. |
| D3 | SPEC-212 Open Question #1 (the shell's global "+ Adicionar ativo" button) | **Resolved, user-accepted: stays a no-op.** Not wired to anything from this screen. |
| D4 | SPEC-212 Open Question #2 (donut vs. spectrum bar for FII sectors) | **Resolved, user-accepted: reuse `AllocationBar`** for both FR-2123 and FR-2124 — no new chart component. |
| D5 | SPEC-212 Open Question #3 (cross-screen cache invalidation) | **Reversed in Phase 5 review.** Originally resolved as "deferred — `refetchOnMount`'s default covers it." `react-correctness-reviewer` proved that assumption wrong: `refetchOnMount` only forces a network refetch when the cached data is already **stale**, and the global `staleTime` is 30s (`app/providers.tsx`) — so a Carteira→Painel SPA navigation within that window silently served pre-mutation figures, undermining BR-2121's core premise. **Implemented**: all six holdings mutations (`lib/portfolio/holdings.ts`) now invalidate `DASHBOARD_KEY` (exported from `lib/dashboard/dashboard.ts`) alongside their own list, via a small shared `invalidateHoldingsAndDashboard` helper. A direct `lib/portfolio` → `lib/dashboard` import is a bit of reaching-across for two peer feature modules; a more scalable cross-feature invalidation strategy (a shared prefix/predicate) is worth revisiting once SPEC-213+ adds more consumers of holdings-derived data (see Post-Implementation Tasks) — not built now to avoid speculative overengineering for consumers that don't exist yet. |
| D6 | FII sector label map exhaustiveness (discovered drafting this plan) | `api/openapi.yaml`'s `fii_sectors[].sector` is declared `type: string` with **no `enum:` constraint** (unlike `indexer_type`, SPEC-109) — so the generated TS type is plain `string`, not a closed union. `lib/dashboard/labels.ts`'s `SECTOR_LABELS` is built as a `Record<string, string>` with a **documented fallback** (an unmapped sector shows its raw value, capitalized) rather than assuming exhaustiveness — a `Record<Sector, string>` (SPEC-211's `INDEXER_LABELS` pattern) isn't achievable here without an `api/openapi.yaml` change, which is out of scope for this spec. |
| D7 | Zero-share allocation entries (FR-2123) | `allocation[]` entries with `share_bps: 0` (Stocks/ETFs, always 0 in the MVP per SPEC-103 D5) are **filtered out before rendering** — a pure client-side display decision over an already-server-sourced list, not hiding real data (the number is still legitimately 0; it's just not worth a zero-width bar segment or a confusing legend row). |

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `app/(app)/dashboard/page.tsx` | Stub → the real Painel screen; its placeholder copy (which incorrectly implies Health Score/Insights are part of this spec) is replaced |

### New Components

| Component | Purpose |
| --------- | ------- |
| `lib/dashboard/dashboard.ts` | `useDashboard()` (`GET /dashboard`, typed from `DashboardResponse`) |
| `lib/dashboard/labels.ts` | `SECTOR_LABELS` (defensive map, D6) |
| `app/(app)/dashboard/summary-hero.tsx` | Hero patrimony card + 3-metric row + stale-ticker notice (FR-2121/2122/2125) |
| `app/(app)/dashboard/allocation-sections.tsx` | Asset-class allocation + FII sector exposure (FR-2123/2124), both via `AllocationBar` |

---

## 6. Implementation Strategy

### Approach

**Incremental**, bottom-up: the data hook + labels first (typed, testable in isolation), then
the two presentational component groups, then the screen that composes them with its
loading/error/empty states, then tests, then closeout. Each phase leaves the `web/` gate green
(`typecheck`/`lint`/`check:api`/`test`/`build`) and is independently reviewable.

### Rollout Method

**Incremental.** The screen replaces a stub behind the existing auth gate; no launch, no flag.

### Rollback Strategy

Revert the `web/**` changes — no backend, no data, no migration involved.

---

## 7. Implementation Phases

### Phase 1 — Data hook & sector labels *(≈ persistence/data)*

#### Tasks
- [x] `lib/dashboard/dashboard.ts` — `useDashboard()`: `GET /dashboard`, typed from
      `components["schemas"]["DashboardResponse"]`, exposing `{dashboard, isLoading, isError,
      refetch}` (mirrors `useFIIHoldings`'s shape). No hand-written DTOs (BR-2124).
- [x] `lib/dashboard/labels.ts` — `SECTOR_LABELS: Record<string, string>` for the six known
      backend sector values (`logistics`/`offices`/`shopping`/`hybrid`/`paper`/`other`, per
      `internal/marketdata/sector.go` — not enforced by the wire type, D6) plus a
      `sectorLabel(sector: string): string` helper falling back to a capitalized raw value for
      anything unmapped, so a future backend sector addition degrades gracefully instead of
      silently rendering blank. `other`'s label is the distinct "Outros / sem cotação" per
      FR-2124, not a generic sector name.

#### Deliverables
- Typed hook + label helper, unit-testable in isolation; the gate stays green.

---

### Phase 2 — Presentational components *(≈ UI vocabulary, reusing existing primitives)*

#### Tasks
- [x] `app/(app)/dashboard/summary-hero.tsx`: the hero (`current_value_centavos` +
      growth badge — `▲`/`text-gain` or `▼`/`text-loss` or neutral at zero, FR-2121), the
      3-metric row (total invested, monthly income, growth again via the **same** shared
      `GrowthFigure` component, D1, FR-2122), and the stale-ticker notice (FR-2125, only
      rendered when `stale_tickers` is non-empty). Growth is explicitly labelled "vs. custo de
      aquisição" — never "no mês" (the design mockup's inaccurate label; the backend tracks no
      time series, per SPEC-212 FR-2121's correction).
- [x] `app/(app)/dashboard/allocation-sections.tsx`: asset-class allocation via `AllocationBar`
      (FR-2123 — entries with `share_bps: 0` filtered out before rendering, D7 — zero-share
      classes never shown as a confusing zero-width segment) and FII sector exposure via
      `AllocationBar` (FR-2124 — rendered only when the FII asset-class slice is non-zero;
      `other` sector labelled distinctly per `sectorLabel`'s fallback). Colors assigned
      positionally from the 5 aurora tokens (cycling by index) — the backend already emits both
      arrays in a fixed, stable order (`compute.go`'s `sectorOrder`, not map iteration), so no
      client-side sort was needed to keep colors stable across fetches.
- [x] Token-styled per Aurora; no raw hex; no new component beyond composing existing ones (D2).

#### Deliverables
- Two components render correctly against representative fixtures (verified in Phase 4's tests
  and live in Phase 3).

---

### Phase 3 — The Painel screen *(≈ application/edge)*

#### Tasks
- [x] `app/(app)/dashboard/page.tsx` — replace the stub: loading skeleton (FR-2127), error +
      retry (FR-2127), the empty-portfolio state (`total_invested_centavos === 0 &&
      current_value_centavos === 0` → a dedicated empty state with a CTA to `/portfolio`,
      FR-2126 — distinct from loading/error), else compose `SummaryHero` +
      `AllocationSections`. **Found and fixed a real bug while wiring this**: the empty state's
      CTA was written as `<Button asChild><a href="/portfolio">...</a></Button>`, but `Button`
      has no Radix `Slot`/`asChild` support at all (confirmed — plain `<button>` wrapper) and
      the codebase's own established pattern for navigation-after-action is `useRouter()`
      (`register`/`login` pages), not composing `Button` with an anchor. Fixed with
      `useRouter().push("/portfolio")` on a plain `onClick`.
- [x] **Live-verified against the real backend** (registered a fresh account via Playwright, not
      a mock): fresh account → the empty state renders, its CTA correctly navigates to
      `/portfolio`; added an FII (HGLG11, 100@R$157,50) + a fixed-income holding (R$5.000,00 @
      10% prefixado) via Carteira, returned to `/dashboard` → hero shows R$21.000,00 patrimony,
      +R$250,00/+1,20% growth (green, ▲); the metric row's growth card shows the **identical**
      figure (D1 proven, not just asserted); asset-class allocation shows FIIs 76,19% / Renda
      fixa 23,81%; FII sector exposure shows Logística 100% as a correct full-width single
      segment. Every number cross-checked arithmetically (100 shares × R$160,00 quoted price −
      R$157,50 cost = exactly the R$250,00 growth shown). Zero console errors.

#### Deliverables
- A working, navigable Painel screen wired to the live backend end to end.

---

### Phase 4 — Testing

#### Unit / Component (Vitest + RTL)
- [x] `lib/dashboard/labels.ts`: every known sector maps to a non-empty pt-BR label; an
      unmapped sector falls back to its capitalized raw value (D6); `other` labelled distinctly.
- [x] `SummaryHero`: a gain fixture (green, `▲`), a loss fixture (red, `▼`), a zero-growth
      fixture (neutral, no arrow); the stale-ticker notice present/absent by fixture; the hero
      badge and the metric-row growth card assert the **identical** formatted string appears
      exactly twice (D1, not two divergent reads); growth labelled "vs. custo de aquisição",
      never "no mês".
- [x] `AllocationSections`: a zero-share class is omitted from the legend; a single-class
      portfolio renders without error; FII sectors omitted entirely for a fixed-income-only
      fixture; the `other` sector's distinct label renders correctly.
- [x] `DashboardPage`: loading → skeleton, no data section rendered; error → retry calls
      refetch; empty portfolio → the dedicated empty state, not a zeroed dashboard; the empty
      state's CTA calls `router.push("/portfolio")` (catches a regression of the Phase 3
      `asChild` bug); populated → both sections render.

#### Integration
- [x] Against a running backend: seeded holdings via Carteira's real create flow, loaded
      `/dashboard`, asserted the rendered figures match the backend's computed response
      (combined with the E2E run below — this repo has no separate integration-test tier for
      the frontend, SPEC-211's precedent).

#### End-to-End (Playwright)
- [x] `e2e/dashboard.spec.ts`: register → the fresh dashboard shows the empty state → its CTA
      navigates to Carteira → add a holding → back on `/dashboard`, the hero reflects the
      non-zero patrimony (asserted `toHaveCount(2)` for the hero + "Total investido" card,
      which legitimately show the identical figure for a same-day zero-accrual holding — not a
      bug, found and correctly diagnosed via the same strict-mode-locator signal that caught
      SPEC-211's real duplicate-CTA bug). Gated to skip without a backend, mirroring
      `e2e/portfolio.spec.ts`. **Run and passing** against the real backend, not just written.

#### Deliverables
- All green in the `web/` CI gate (131/131 tests, 17 files) + a clean production `build`; E2E
  run and passing against the real backend.

---

### Phase 5 — Documentation & Closeout

#### Tasks
- [x] **CHANGELOG** `[Unreleased]` entry.
- [x] **No `api/openapi.yaml` change** — confirmed via `git diff main -- api/openapi.yaml` (empty).
- [x] Flip **SPEC-212 + PLAN-212 → Done**; update the specs/plans indexes.
- [x] **Review** with **frontend-reviewer** + **react-correctness-reviewer**.
      **`frontend-reviewer`: PASS**, two non-blocking notes, one applied: the two
      `AllocationSections` card titles used `<h3>` with no `<h2>` anywhere on the page (the
      shell's `TopBar` renders the page `<h1>`) — bumped to `<h2>` to match
      `fii-table.tsx`/`fixed-income-table.tsx`'s precedent. The second note (`aria-live` on
      loading/error) was **not** applied — no sibling screen has it either, so a one-screen
      addition would read as inconsistent rather than an improvement (see Post-Implementation
      Tasks for the systemic follow-up).
      **`react-correctness-reviewer`: CHANGES REQUESTED → fixed → re-verified clean.** One real,
      significant bug: none of the six holdings mutations (`lib/portfolio/holdings.ts`,
      SPEC-211) invalidated the Dashboard's query cache, and the global `staleTime` is 30s
      (`app/providers.tsx`) — so a Carteira→Painel **SPA navigation** (not a full reload) within
      that window silently served pre-mutation figures, directly undermining BR-2121's "every
      figure is authoritative" premise. This also reverses this plan's own D5 (originally
      "deferred, `refetchOnMount` covers it" — that assumption was wrong). Fixed: `DASHBOARD_KEY`
      exported from `lib/dashboard/dashboard.ts`, all six mutations now invalidate it alongside
      their own list via a shared `invalidateHoldingsAndDashboard` helper. Added **new** direct
      hook tests (`lib/portfolio/holdings.test.tsx` — the existing SPEC-211 component tests mock
      this module wholesale and never actually exercised the real invalidation logic) proving the
      fix for both `onSuccess` and `onSettled` mutation paths, across both FII and fixed-income
      resources. Also **strengthened** `e2e/dashboard.spec.ts` to navigate back to `/dashboard`
      via a real sidebar click (`getByRole("link", { name: "Painel" })`) instead of
      `page.goto()` — the reviewer correctly noted the original E2E test's full-reload navigation
      accidentally masked the bug (a fresh page load always gets a fresh `QueryClient`, so it
      never exercises the SPA-cache staleness path a real user would hit). Re-ran the E2E test
      against the real backend after the fix: passes, proving the invalidation actually works
      live, not just in a mocked unit test.
- [x] **PT-BR lesson** `docs/lessons/SPEC-212-aula.html` via **frontend-lesson-writer**
      (product-focused).

#### Deliverables
- Docs updated, spec closed, lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| The unmapped-sector fallback (D6) never actually gets exercised until the backend adds a 7th sector | Low | Unit-tested directly against a synthetic unmapped value, not left to chance discovery |
| Growth-figure duplication (hero + metric card) drifts if a future edit changes one but not the other | Low | D1's shared-value discipline + a test asserting both render the identical figure from one fixture |
| Empty-portfolio detection (`both totals === 0`) misfires for a real edge case (e.g. a FII gifted at zero cost basis, nonzero current value) | Low | The condition requires **both** totals to be zero — a nonzero current value with zero cost basis does not trigger it; covered in SPEC-212 §9 Edge Cases and worth a regression test if it proves fragile |
| Scope creep into Health Score / Insights, since the current stub's copy already (incorrectly) implies they're part of this spec | Low | SPEC-212 §2 Scope is explicit; Phase 3 replaces the stub's copy along with the rest of it |
| **(materialized, fixed)** The empty state's CTA was written as `Button asChild` wrapping an anchor — `Button` has no `Slot`/`asChild` support at all | Low | Found wiring Phase 3, before it ever shipped. Fixed with `useRouter().push(...)` on a plain `onClick`, matching the codebase's own established navigation-after-action pattern (`register`/`login` pages) |
| **(materialized, fixed — High, not Low)** No holdings mutation invalidated the Dashboard's query cache; a 30s global `staleTime` meant an SPA navigation back to Painel after editing Carteira could silently show pre-mutation figures | High | Caught by `react-correctness-reviewer` in Phase 5, not by the original D5 assumption or by the E2E test as first written (which used `page.goto()`, a full reload, accidentally sidestepping the bug). Fixed with cross-feature cache invalidation (D5, reversed); proven with new direct hook tests plus a strengthened E2E test that actually performs an SPA navigation |

---

## 9. Validation Checklist

### Functional Validation
- [x] FR-2121…FR-2128 implemented; Epic 3 acceptance criteria satisfied.
- [x] BR-2121…BR-2126 respected (read-only/no-client-computation, integer money display-only,
      no AI guards needed, generated types, visible degradation, identity from session).

### Technical Validation
- [x] Consumes SPEC-103 only; **no `api/openapi.yaml` change**; `check:api` drift guard green.
- [x] `401`→login handled; no `user_id` on the wire; no float, no order affordance.
- [x] No new runtime dependency; no new UI primitive (D2/D4).

### Quality Validation
- [x] Vitest/RTL (134 tests) + the combined integration/E2E run passing.
- [x] a11y (AA contrast on gain/loss text; no color-only signal — the `▲`/`▼` glyphs carry the
      direction too, not just color; heading levels corrected to `<h2>` per Phase 5 review).
- [x] Reviewed by **frontend-reviewer** (PASS) + **react-correctness-reviewer** (CHANGES
      REQUESTED → the cross-screen cache-invalidation bug fixed → re-verified clean); docs
      updated.

---

## 10. Definition of Done

- [x] All phases complete; SPEC-212 acceptance criteria satisfied.
- [x] Unit/component (134 tests) + the combined integration/E2E run green; `web/` gate
      (`typecheck`/`lint`/`test`/`check:api`/`build`) all clean.
- [x] **CHANGELOG** updated; **`api/openapi.yaml` unchanged** (asserted via `git diff`).
- [x] **SPEC-212 + PLAN-212 flipped to Done**; specs/plans indexes updated.
- [x] **PT-BR lesson** `docs/lessons/SPEC-212-aula.html` produced (via **frontend-lesson-writer**).
- [x] Reviewed by the frontend review agents (one real bug found and fixed — the cross-screen
      cache-invalidation gap — re-verified clean after the fix).
- [ ] Pull Request approved.

---

## 11. Deliverables

### Code Deliverables
- `lib/dashboard/{dashboard,labels}.ts`, `app/(app)/dashboard/{summary-hero,allocation-sections,page}.tsx`,
  the cross-feature invalidation fix in `lib/portfolio/holdings.ts`, and their tests (incl. the
  new `lib/portfolio/holdings.test.tsx`).

### Documentation Deliverables
- CHANGELOG entry, PT-BR lesson, specs/plans index updates.

---

## 12. Post-Implementation Tasks

### Future Improvements
- A more scalable cross-feature cache-invalidation strategy (a shared prefix/predicate instead of
  each holdings mutation explicitly knowing about `DASHBOARD_KEY`) once SPEC-213+ adds more
  consumers of holdings-derived data — D5's direct-import fix is right-sized for one consumer,
  not built to anticipate more.
- A donut-chart upgrade for FII sector exposure (Open Question #2 / D4) if the spectrum bar
  proves visually insufficient once real users see it.
- Wiring the shell's "+ Adicionar ativo" button (Open Question #1 / D3), once some screen
  actually claims it.
- An `api/openapi.yaml` `enum:` constraint on `fii_sectors[].sector` (D6) — a small backend-side
  contract precision improvement, out of scope for this frontend spec.
- A systemic `aria-live`/`role="status"` pass for loading/error states across the whole app
  (`frontend-reviewer` non-blocking note) — not added to this screen alone, since no sibling
  screen (Carteira, Perfil) has it either; a one-screen addition would read as inconsistent
  rather than an improvement. Worth its own cross-cutting pass.

### Technical Debt
- None anticipated beyond D6's documented fallback, which is a deliberate defensive design
  choice, not a workaround.
