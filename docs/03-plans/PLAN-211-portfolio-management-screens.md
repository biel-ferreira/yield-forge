# PLAN-211 — Portfolio Management Screens

## 1. Document Information

| Field           | Value                                   |
| --------------- | --------------------------------------- |
| Plan Name       | Portfolio Management Screens            |
| Related Feature | Portfolio Management Screens (Carteira) |
| Related Spec    | [SPEC-211](../02-specs/SPEC-211-portfolio-management-screens.md) (Approved) |
| Version         | 0.1.0                                    |
| Status          | Draft                                    |
| Author          | Gabigol                                  |
| Last Updated    | 2026-07-02                               |

> **Phase-order note.** Frontend spec — the template's backend phase order is mapped to its
> frontend analogue (data → components → screens → composition → tests → docs), the same
> mapping PLAN-210 used. This feature **builds on SPEC-200** (typed client, design system,
> shell, auth gate, Vitest/Playwright) and **reuses SPEC-210's `segmented.tsx`** directly — but
> it is meaningfully bigger than PLAN-210: two independent CRUD verticals (FII, fixed income)
> and a genuinely new UI primitive (the add/edit/delete modal), so it gets more phases.

---

## 2. Objective

### Goal

Turn the `/portfolio` stub into the real **Carteira** screen (SPEC-211): list, add, edit, and
delete FII and fixed-income holdings against the live SPEC-102 + SPEC-109 backend, using the
SPEC-200 typed client and the Aurora design system.

### Expected Outcome

A user can register their full portfolio — FIIs and fixed-income positions, including the SPEC-109
rate indexer (`prefixado` / `% do CDI` / `IPCA+`) — see it listed with the resolved effective rate
and its reference date, and edit or remove any holding, all without a page reload and all validated
at the edge before ever reaching the wire. No new endpoint, **no `api/openapi.yaml` change**.

---

## 3. Scope

### Included

- **Regenerating `web/lib/api/schema.ts`** — currently stale vs. `api/openapi.yaml` (SPEC-109's
  `indexer_type` / `effective_annual_rate_bps` / `MarketIndicatorResponse` additions were never
  regenerated into the frontend types; `npm run check:api` fails on `main` right now). This has
  to happen before anything else in this plan can type-check.
- **Data hooks** for both holding types (`lib/portfolio/holdings.ts`) and market indicators
  (`lib/portfolio/market.ts`), plus enum ↔ pt-BR label maps (`lib/portfolio/labels.ts`).
- **pt-BR money/rate input parsing** (`lib/money.ts` extension, FR-2119) — the inverse of the
  existing display formatters.
- A new, reusable **modal dialog primitive** (`components/ui/dialog.tsx`) and a **confirm-dialog**
  built on it — SPEC-211 is the first screen needing a modal.
- The **FII section** (list, add/edit modal, delete) and the **fixed-income section** (list,
  add/edit modal with the indexer picker + live reference display, delete).
- The **Carteira screen** composing both sections.
- Tests (Vitest/RTL + integration + a gated Playwright E2E) and the SDD closeout.

### Excluded

- Any **backend change** — SPEC-102 and SPEC-109 are both Done; this consumes them. No new
  endpoint, no `openapi.yaml` edit.
- **Current value / unrealized gain-loss** computation — cost basis only (BR-2113); that's the
  Dashboard's job (SPEC-212).
- **Ticker autocomplete** against the marketdata universe (SPEC-211 §14 Q3, resolved: free-text
  for the MVP — see D3 below).
- A **staleness threshold/badge** on the resolved effective rate beyond showing its raw
  `reference_date` (SPEC-211 FR-2120, resolved: kept deliberately simple for the MVP).

---

## 4. Dependencies

### Technical Dependencies

- **SPEC-102** backend (8 holdings endpoints, `FIIHoldingRequest/Response`,
  `FixedIncomeRequest/Response`) — Done and running.
- **SPEC-109** backend (`indexer_type`, `effective_annual_rate_bps`, `GET /market/indicators`,
  `MarketIndicatorResponse`) — Done and running, but **not yet reflected in the frontend's
  generated types** (see Scope above — the first task of Phase 1).
- **SPEC-200** foundation: the typed client, TanStack Query, the `(app)` shell + `RequireAuth`,
  the Aurora design system, `components/ui/{segmented,input,badge,empty-state}.tsx`, the
  Vitest/Playwright setup, `lib/money.ts`'s display formatters.
- **SPEC-210** precedent: `components/ui/segmented.tsx` is reused as-is for both the indexer and
  liquidity single-selects (already generic, already proven for a 3-option set); the
  `lib/<feature>/{labels,hooks}.ts` module shape and the 404→`null` query pattern are mirrored.

### External Dependencies

- None new. Zero new runtime deps (ADR-0003): the modal is built on the **native `<dialog>`
  element** (see D1), not a Radix/shadcn import — mirroring PLAN-210 D5's native-primitive
  choice for the slider.

### Blocking Decisions

| # | Decision | Resolution (this plan) |
|---|----------|------------------------|
| D1 | Add/edit UI pattern (SPEC-211 §14 Q1) | **Modal dialog**, built on the **native `<dialog>` element** (`showModal()`/`close()`) — modern evergreen browsers give free focus-trap, Escape-to-close, and top-layer backdrop blocking for it, so it needs no new dependency (mirrors PLAN-210 D5's "native over library" call for the slider). One `components/ui/dialog.tsx` primitive, reused for both add and edit (prefilled) on both holding types. |
| D2 | Delete confirmation UX (SPEC-211 §14 Q2) | **Confirm dialog**, `components/ui/confirm-dialog.tsx` composed on top of D1's `dialog.tsx` — simplest, safest default; reused for both FII and fixed-income deletes. |
| D3 | Ticker input (SPEC-211 §14 Q3) | **Free text, uppercase-normalized client-side.** Matches the backend, which does not validate a ticker against a known universe at creation (SPEC-102) — a malformed ticker is still caught by the existing `400` (`ErrInvalidTicker`). No new lookup endpoint. |
| D4 | Indexer & liquidity single-selects | **Reuse `components/ui/segmented.tsx` as-is** — already generic, already proven for a 3-option set (risk profile, SPEC-210). No new component. |
| D5 | Money/rate input widget (FR-2119) | A plain `components/ui/input.tsx` text field; parsing (`parseCentavos`/`parseBps`, new in `lib/money.ts`) happens on blur/submit, not live-as-you-type masking — deliberately simple for the MVP, mirrors the FR-2120 staleness-display simplicity call. |
| D6 | Maturity date input | Native `<input type="date">` — zero-dep, consistent with D1/D5's native-primitive bias. |
| D7 | Live reference + reference-date display (FR-2120) | Plain presence/absence check (`GET /market/indicators` has the indicator or it doesn't) plus the raw `reference_date` shown next to the resolved rate — **no computed staleness threshold** (CDI/SELIC update ~daily, IPCA ~monthly; a single threshold would misjudge one of them). Resolved this session, already folded into SPEC-211 FR-2120. |
| D8 | Data hook module location | `lib/portfolio/` (mirrors `lib/profile/`): `holdings.ts` (FII + FI CRUD hooks), `market.ts` (`useMarketIndicators` + an indicator-lookup helper), `labels.ts` (indexer/liquidity/indicator pt-BR maps). |
| D9 | Stale generated types (discovered this session) | `npm run gen:api` + commit is **Phase 1, Task 1** — a hard prerequisite, not an incidental cleanup; nothing past it type-checks against the real SPEC-109 contract otherwise. |

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `web/lib/api/schema.ts` | Regenerated from `api/openapi.yaml` (D9) — picks up `indexer_type`, `effective_annual_rate_bps`, `MarketIndicatorResponse` |
| `app/(app)/portfolio/page.tsx` | Stub → the real Carteira screen |
| `docs/05-design/design-system.md` | Optional: note the `dialog`/`confirm-dialog` primitives alongside the existing component inventory |

### New Components

| Component | Purpose |
| --------- | ------- |
| `lib/portfolio/holdings.ts` | `useFIIHoldings`/`useCreate/Update/DeleteFIIHolding`, `useFixedIncomeHoldings`/`useCreate/Update/DeleteFixedIncomeHolding` |
| `lib/portfolio/market.ts` | `useMarketIndicators` (`GET /market/indicators`) + a pure `findIndicator(list, indicator)` lookup helper |
| `lib/portfolio/labels.ts` | `Indexer`/`LiquidityType`/`Indicator` enum ↔ pt-BR label maps |
| `lib/money.ts` (extended) | `parseCentavos`/`parseBps` — pt-BR string → integer (FR-2119) |
| `components/ui/dialog.tsx` | Native-`<dialog>`-backed modal primitive (D1) |
| `components/ui/confirm-dialog.tsx` | Delete confirmation, composed on `dialog.tsx` (D2) |
| `app/(app)/portfolio/fii-table.tsx`, `fii-form.tsx` | FII list + add/edit modal form |
| `app/(app)/portfolio/fixed-income-table.tsx`, `fixed-income-form.tsx` | FI list (incl. resolved rate + reference date) + add/edit modal form (indexer picker + live reference) |

---

## 6. Implementation Strategy

### Approach

**Incremental**, bottom-up: fix the stale types first (nothing else compiles otherwise), then data
hooks + parsing (typed, testable in isolation), then the new modal primitive, then the two CRUD
verticals (FII first — simpler, proves the pattern; fixed income second — reuses it, adds the
indexer/reference complexity), then the screen that composes them, then tests, then closeout. Each
phase leaves the `web/` gate green (`typecheck`/`lint`/`check:api`/`test`/`build`) and is
independently reviewable.

### Rollout Method

**Incremental.** The screen replaces a stub behind the existing auth gate; no launch, no flag.

### Rollback Strategy

Revert the `web/**` changes (plus the `lib/api/schema.ts` regen, which is backend-contract-neutral
— it only adds types for endpoints that already exist) — no backend, no data, no migration
involved.

---

## 7. Implementation Phases

### Phase 1 — Fix stale types; data hooks & input parsing *(≈ persistence/data)*

#### Tasks
- [x] **`npm run gen:api` and commit the result** (D9) — closes the current `check:api` drift
      before any SPEC-109-shaped code is written.
- [x] `lib/portfolio/labels.ts` — `Indexer` (`prefixado`/`cdi_percentual`/`ipca_spread`),
      `LiquidityType`, and `Indicator` (`selic`/`cdi`/`ipca`) enum ↔ pt-BR label maps, typed from
      `components["schemas"]` (mirrors `lib/profile/labels.ts`). Also added `referenceIndicator`
      (`Indexer` → `Indicator | null`), used by Phase 4's live-reference display.
- [x] `lib/money.ts`: `parseCentavos(input: string): number | null` and
      `parseBps(input: string): number | null` (FR-2119) — `"1.234,56"` → `123456`,
      `"10,5"` → `1050`; `null` on malformed/empty-when-required input (never coerced to `0`).
- [x] `lib/portfolio/holdings.ts` — `useFIIHoldings()` (list) + `useCreateFIIHolding()` /
      `useUpdateFIIHolding()` / `useDeleteFIIHolding()`; the fixed-income mirror
      (`useFixedIncomeHoldings()` + create/update/delete). List query invalidated on any mutation's
      success (TanStack Query). No hand-written DTOs (BR-2116). Update/delete throw a new
      `ApiError(status, message)` (`lib/api/error.ts`, extended) so Phase 3/4 form components can
      distinguish a `404` (list-refresh, BR-2111) from a `400` (inline message) without string-sniffing.
- [x] `lib/portfolio/market.ts` — `useMarketIndicators()` (`GET /market/indicators`) +
      `findIndicator(list, indicator)` pure helper (used by both the live-reference display and the
      table's per-holding reference lookup, FR-2120/D7).

#### Deliverables
- Typed hooks + parsing + labels, unit-testable in isolation; the gate stays green
  (`check:api` passes for the first time against the real SPEC-109 contract).

---

### Phase 2 — Modal & confirm-dialog primitives *(≈ new UI vocabulary)*

#### Tasks
- [x] `components/ui/dialog.tsx` (D1) — wraps the native `<dialog>` element: `open`/`onClose`
      props, imperative `showModal()`/`close()` sync via `useEffect`, backdrop-click-to-close,
      `aria-labelledby` wired to a required `title` prop (binding-guard-style: a dialog without a
      title is unrepresentable in the prop contract, mirroring `InsightCard`'s required
      `explanation`).
- [x] `components/ui/confirm-dialog.tsx` (D2) — built on `dialog.tsx`: `title`, `description`,
      `onConfirm`, `onCancel`, destructive styling per Aurora tokens. Added a new `destructive`
      `Button` variant (`border-loss/50 bg-loss/5 text-loss`) — mirrors `Badge`'s existing
      soft-tint pattern for semantic colors rather than inventing a separate saturated "danger"
      brand color (CLAUDE.md reserves gain/loss/caution/info as figure colors, never a solid fill).
- [x] Token-styled per the Aurora design system; no raw hex; reduced-motion respected (global
      `prefers-reduced-motion` media query already covers it, no per-component work needed).
- [x] **Live-verified in a browser** (Playwright against the dev server, not just typecheck):
      both dialogs open centered with backdrop, backdrop-click and the X button both close them,
      zero console errors. Screenshots reviewed, not committed (scratch verification only).

#### Deliverables
- Two reusable, tested primitives; render in the styleguide for a visual check.

---

### Phase 3 — FII section *(≈ first CRUD vertical, proves the pattern)*

#### Tasks
- [x] `app/(app)/portfolio/fii-table.tsx` — populated table (ticker, quantity, average price,
      actions) sorted by ticker; empty state ("nenhum FII cadastrado" + CTA); loading skeleton;
      transient-error retry (FR-2111). Exports `FiiSection`, owning the list + modal orchestration
      end to end (mirrors `app/(app)/profile/page.tsx`'s single-file load+form ownership).
- [x] `app/(app)/portfolio/fii-form.tsx` — the D1 modal, add and edit (prefilled) sharing one
      component: ticker (free text, uppercase-normalized, D3), quantity (positive whole number,
      ≥1), average price (pt-BR input, `parseCentavos`, ≥0) (FR-2112/2113).
- [x] Delete wired through `confirm-dialog.tsx` (FR-2114). Extended `ConfirmDialog` with an
      optional `error` slot for a genuine delete failure (not the 404 case).
- [x] A mutation `404` (edit/delete) refreshes the list rather than erroring (BR-2111); a `400`
      surfaces the server message inline with input preserved. Required switching the update/delete
      hooks (Phase 1) from `onSuccess` to `onSettled` invalidation — a 404 doesn't fire `onSuccess`
      but the cached list is still stale and must refresh; discovered while wiring this phase.
- [x] **Live-verified against the real backend** (registered a fresh account via Playwright,
      not a mock): create → row appears correctly formatted; edit → prefill matches exactly,
      quantity update reflects; delete → confirm dialog → row removed, empty state returns.
      Zero console errors. `app/(app)/portfolio/page.tsx` now renders `FiiSection` for real
      (not a throwaway harness) — Phase 5 adds the fixed-income section alongside it.

#### Deliverables
- A working FII section: list/add/edit/delete against the live backend.

---

### Phase 4 — Fixed-income section *(≈ second CRUD vertical, adds the indexer)*

#### Tasks
- [x] `app/(app)/portfolio/fixed-income-table.tsx` — populated table (name, institution, invested
      amount, resolved effective rate + `reference_date` or "sem referência disponível", liquidity
      label, pt-BR maturity date or "—") (FR-2115/2120/D7); empty/loading/error states. Added
      `lib/date.ts` (`formatDateBR`/`todayISO`) — the render-edge date helper this and the form need.
- [x] `app/(app)/portfolio/fixed-income-form.tsx` — the D1 modal: name/institution (required),
      invested amount (`parseCentavos`, ≥1), **indexer** single-select (reuses `segmented.tsx`,
      D4) whose selection changes the rate-value input's label/unit, rate value (`parseBps`, ≥0),
      **liquidity** single-select (reuses `segmented.tsx`, D4) — choosing Diária clears/disables
      maturity (`null`), choosing No vencimento requires one; maturity date (D6, past-date
      rejected at the edge for a new at-maturity holding, edit-mode exempt per the backend's
      create-time-only rule) (FR-2116/2117).
- [x] Live reference display, inline with the indexer picker: when % do CDI / IPCA+ is selected,
      show `"CDI atual: 10,50% a.a. (ref. 01/07/2026)"` (or the IPCA equivalent) via
      `useMarketIndicators` + `findIndicator`; "indisponível no momento" when absent/loading, never
      blocking the save path (FR-2120).
- [x] Delete wired through `confirm-dialog.tsx` (FR-2118).
- [x] **Bug found and fixed during live verification:** `FiiForm`/`FixedIncomeForm` are always
      mounted (only `open` toggles the native `<dialog>`), so `useState`'s initializer — which
      only runs on first mount — never re-ran when switching edit targets or reopening "add".
      Reopening "add" after an abandoned attempt (or editing a different holding after closing a
      previous edit) leaked the prior session's stale form state. Fixed with a `key` prop
      (`"closed"` / `"add"` / the holding's id) on both forms forcing a fresh mount per target;
      regression-verified live (reopening "add" after an edit session now shows a genuinely blank
      form). Also fixed a `Dialog` `className` bug found in the same pass: it was merged onto the
      outer `<dialog>` element, but the visible width comes from the inner content `<div>`'s
      hardcoded `max-w-md` — so a consumer's `className="max-w-lg"` silently did nothing.
- [x] **Live-verified against the real backend**, including the actual live-reference display
      resolving a genuine seeded CDI/IPCA value (not a mock): create a `cdi_percentual` holding
      (120% do CDI, CDI=10,50%) → table shows "% do CDI · 12,60%"; edit to `ipca_spread`
      (+5,80%, IPCA=10,50%) → table shows "IPCA + · 16,30%" — both effective-rate computations
      match the backend's math exactly. Zero console errors.

#### Deliverables
- A working fixed-income section: list/add/edit/delete, indexer picker, live reference display,
  against the live backend.

---

### Phase 5 — The Carteira screen *(≈ composition)*

#### Tasks
- [x] `app/(app)/portfolio/page.tsx` — compose the FII section + fixed-income section, page
      heading, "Adicionar FII" / "Adicionar renda fixa" CTAs opening their respective modals.
      No redundant top-level heading added: the shell's `TopBar` already renders the page-level
      "Carteira" title (route-derived, SPEC-200), so the page just stacks the two sections, each
      owning its own h2 + CTA — documented in a code comment so this isn't mistaken for an
      oversight later. Not width-constrained (unlike the narrower Perfil form) — the
      fixed-income table has 7 columns and needs the room.
- [x] Verified both sections' empty states render independently (not a combined blank page) when
      the portfolio is entirely empty — **live-verified** (fresh account, both "Nenhum FII
      cadastrado" / "Nenhuma renda fixa cadastrada" render side by side, correctly separate).
      Also live-verified both sections **populated simultaneously** (an FII + a fixed-income
      holding together) — clean layout at a normal viewport width, zero console errors.

#### Deliverables
- A working, navigable Carteira screen wired to the live backend end to end.

---

### Phase 6 — Testing

#### Unit / Component (Vitest + RTL)
- [x] `parseCentavos`/`parseBps` (FR-2119): table-tested (`lib/money.test.ts`), incl.
      malformed-input rejection and the round-trip with `formatCentavos`/`formatBps`.
- [x] Validation gating for both forms (`fii-form.test.tsx`, `fixed-income-form.test.tsx`): FII
      ticker/quantity/price; FI required fields, liquidity ↔ maturity interaction, past-date
      rejection (create-only, per FR-2116's edit-mode exemption — tested explicitly).
- [x] Empty vs. populated list rendering for both sections (`fii-table.test.tsx`,
      `fixed-income-table.test.tsx`).
- [x] Submit builds the exact request body for each resource (no `user_id`); a `404` on edit/delete
      is handled as list-refresh, not an error toast — both tested via a hand-written
      `mutationHook<TData,TVariables>` fake (generic over create/update/delete's differing
      TanStack Query shapes; no mocking library, per convention).
- [x] Indexer selection: the rate-value label/unit adapts per indexer; edit prefills the correct
      indexer + value; the live-reference display renders the fetched value/date and degrades to
      "indisponível" without blocking save (`fixed-income-form.test.tsx`).
- [x] Effective-rate reference date (D7): the table shows `reference_date` when the indicator is
      present, "sem referência disponível" when absent — both indicators (CDI/IPCA) and
      `prefixado` (never shows a reference) (`fixed-income-table.test.tsx`).
- [x] `dialog.tsx`/`confirm-dialog.tsx`: open/close, close-button, backdrop-click, title/children
      rendering, required-confirmation, pending-disables-actions, inline error. **Scope note:**
      jsdom has no `<dialog>` implementation at all (confirmed empirically — `showModal`/`close`
      are simply absent); added a minimal polyfill (`vitest.setup.ts`) so the component's own
      control logic is testable, but Escape-to-close and real focus-trap/return are native
      browser guarantees jsdom cannot emulate — those were proven in this session's live-browser
      (Playwright) verification during Phases 2–5, not re-provable here.

#### Integration
- [x] Against a running backend: create → list reflects it → edit → list reflects it → delete →
      list no longer has it, for both FII and fixed-income. **Combined with E2E below** — this
      repo has no separate integration-test tier for the frontend (confirmed: no prior spec built
      one either); the same real-network Playwright run mirrors `e2e/profile.spec.ts`'s precedent.

#### End-to-End (Playwright)
- [x] `e2e/portfolio.spec.ts`: add an FII holding and a fixed-income holding, see both appear;
      edit the FII holding, the list reflects it; delete it, confirm it's gone (the fixed-income
      one remains). **Actually run against the real backend** (Go API + Postgres), not just
      written — caught and fixed two real bugs in the process (see Risks): the empty-state CTA
      duplication, and non-unique "Editar"/"Excluir" accessible names once both sections have
      rows (fixed by row-scoping the test, not by weakening it).

#### Deliverables
- All green in the `web/` CI gate: `typecheck` ✅ `lint` ✅ `test` (95/95, 13 files) ✅ `check:api`
  ✅. E2E (`e2e/portfolio.spec.ts`) run and passing against the real backend — not CI-gated
  (matches `e2e/profile.spec.ts`'s existing precedent, documented in `playwright.config.ts`).

---

### Phase 7 — Documentation & Closeout

#### Tasks
- [ ] **CHANGELOG** `[Unreleased]` entry.
- [ ] **No `api/openapi.yaml` change** — assert it (consumes SPEC-102 + SPEC-109; adds no
      endpoint). Note the `lib/api/schema.ts` regen (D9) in the entry — it's a types-only fix, not
      a contract change.
- [ ] Flip **SPEC-211 + PLAN-211 → Done**; update the specs/plans indexes.
- [ ] Optional: note `dialog`/`confirm-dialog` in `design-system.md`.
- [ ] **Review** with **frontend-reviewer** + **react-correctness-reviewer**; fix blockers.
- [ ] **PT-BR lesson** `docs/lessons/SPEC-211-aula.html` via **frontend-lesson-writer**
      (product-focused).

#### Deliverables
- Docs updated, spec closed, lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Stale `lib/api/schema.ts` (D9) blocks everything downstream | High (but trivial) | Sequenced as Phase 1's literal first task; `check:api` proves it's fixed before any other code lands |
| Native `<dialog>` focus-trap/backdrop correctness across browsers | Low | Evergreen-browser baseline feature; verified by the frontend-reviewer a11y check + RTL open/close/focus-return tests |
| Two independent CRUD verticals (FII, fixed income) risk duplicated table/modal-form logic | Medium | Accept the duplication for the MVP (two verticals, not five) rather than force a premature generic-CRUD abstraction; revisit only if a third vertical appears |
| FR-2120's indicator cross-reference (`indexer_type` → which `Indicator` to look up) has an edge case for `prefixado` (no indicator at all) | Low | `findIndicator` is a small pure helper, unit-tested including the `prefixado`/no-lookup case |
| Scope creep into the Dashboard's current-value computation | Low | BR-2113 draws a hard line (cost basis only); enforced by review |
| **(materialized, fixed)** `FiiForm`/`FixedIncomeForm` never remounted between edit targets (same JSX position, only `open` toggled) — `useState`'s initializer only runs once, so switching edit targets or reopening "add" leaked the previous session's stale form state | Medium | Found live-verifying Phase 4 (a Playwright script's ambiguous selector surfaced it). Fixed with a `key` per target (`formTarget.id`/`"add"`/`"closed"`) forcing a clean remount — the standard React "key resets state" pattern |
| **(materialized, fixed)** Each section's empty state showed two identically-labeled "Adicionar…" buttons (header CTA + `EmptyState`'s own CTA) — an a11y/testing ambiguity, not just cosmetic duplication | Low | Found writing the E2E test (`getByRole` strict-mode violation). Fixed by hiding the header CTA specifically when the empty state is showing (`isEmpty` gate) — mutually exclusive by construction, not by test-side workaround |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-2111…FR-2120 implemented; Epic 1 acceptance criteria satisfied for both holding types.
- [ ] BR-2111…BR-2117 respected (ownership-as-404, money bidirectional-integer, cost-basis-only,
      edge validation, no AI guards needed, generated types, reference rates never
      client-computed).

### Technical Validation
- [ ] Consumes SPEC-102 + SPEC-109 only; **no `api/openapi.yaml` change**; `check:api` drift
      guard green (proves D9 is resolved and stays resolved).
- [ ] `404`→list-refresh and `401`→login handled; no `user_id` on the wire; no float, no order
      affordance.
- [ ] No new runtime dependency (D1/D5/D6 all native-primitive choices).

### Quality Validation
- [ ] Vitest/RTL + integration + gated E2E passing.
- [ ] a11y (dialog focus trap/Escape/labelling; AA contrast); reduced-motion respected.
- [ ] Reviewed by **frontend-reviewer** + **react-correctness-reviewer**; docs updated.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-211 acceptance criteria satisfied.
- [ ] `lib/api/schema.ts` regenerated and in lockstep with `api/openapi.yaml` (`check:api` green).
- [ ] Unit/component + integration + gated E2E green in the `web/` CI gate.
- [ ] **CHANGELOG** updated; **`api/openapi.yaml` unchanged** (asserted).
- [ ] **SPEC-211 + PLAN-211 flipped to Done**; specs/plans indexes updated.
- [ ] **PT-BR lesson** `docs/lessons/SPEC-211-aula.html` produced (via **frontend-lesson-writer**).
- [ ] Reviewed by the frontend review agents.
- [ ] Pull Request approved.

---

## 11. Deliverables

### Code Deliverables
- `lib/portfolio/{holdings,market,labels}.ts`, `lib/money.ts` (extended), `components/ui/{dialog,confirm-dialog}.tsx`,
  the FII and fixed-income table/form components, the Carteira screen, and their tests.

### Documentation Deliverables
- CHANGELOG entry, PT-BR lesson, specs/plans index updates; optional `design-system.md` note.

---

## 12. Post-Implementation Tasks

### Future Improvements
- Ticker **autocomplete** against the marketdata universe (SPEC-211 §14 Q3), if a lookup endpoint
  is ever justified for another feature too.
- A shared generic CRUD-modal abstraction, if a third holding-like vertical appears (SPEC-212+
  don't currently need one — they're read-only).
- A computed staleness indicator on the resolved effective rate (D7's simpler MVP treatment could
  grow a per-indicator threshold later if user feedback asks for it).

### Technical Debt
- None anticipated beyond the now-resolved D9 (stale generated types), which this plan closes as
  its first task.
