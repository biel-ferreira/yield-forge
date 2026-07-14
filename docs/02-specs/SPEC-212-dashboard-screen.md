# SPEC-212 — Dashboard Screen (Painel)

## 1. Document Information

| Field        | Value                                                                 |
| ------------ | ---------------------------------------------------------------------- |
| Feature Name | Dashboard Screen (Painel)                                              |
| Feature ID   | SPEC-212                                                                |
| Version      | 0.1.0                                                                   |
| Status       | Approved                                                                |
| Author       | Gabigol                                                                |
| Last Updated | 2026-07-13                                                              |
| Related PRD  | [Epic 3](../01-product/PRD.md) (FR-004 Portfolio Summary, FR-005 Allocation Breakdown) |
| Consumes     | [SPEC-103](SPEC-103-dashboard.md) (backend) over the [OpenAPI contract](../../api/openapi.yaml); built on [SPEC-200](SPEC-200-app-foundation.md); reuses the `AllocationBar` component (SPEC-200) and the Aurora design system ([ADR-0006](../04-architecture/adr/ADR-0006-frontend-ui-stack-and-design-system.md)) |

---

## 2. Overview

### Purpose

Turn the `/dashboard` stub into the real **Painel** screen — the frontend face of SPEC-103's
`GET /dashboard`. A single read-only view answering "where do I stand right now": total
patrimony, invested cost basis, monthly passive income, growth vs. cost basis, allocation by
asset class, and FII sector exposure. It is the **first screen the investor lands on** (the
sidebar's default route, per the SPEC-200 shell) and the one every other frontend screen
(Carteira for editing, Insights/Health/Projections for reasoning over the same facts) links back
to.

### Business Value

"Understand my totals, allocation, and sector exposure at a glance" (PRD Epic 3) is the
product's core read experience — the payoff for having gone through SPEC-210 (profile) and
SPEC-211 (holdings). Nothing here is computed client-side: every figure is read verbatim from
the backend's deterministic computation (SPEC-103 BR-1031, "facts are computed, not
generated" applied to the read model), so the screen's only job is **faithful, legible
display** — pt-BR formatting, reconciling visuals, honest degradation — never arithmetic.

### Scope

**In scope**
- The hero patrimony card (current value + growth vs. cost basis), a three-metric row (total
  invested, monthly passive income, growth), asset-class allocation, and FII sector exposure —
  the full `GET /dashboard` payload (FR-004/FR-005), and only that payload.
- Loading, error, empty-portfolio, and stale-quote-degradation states.
- One new data hook (`lib/dashboard/dashboard.ts`) and the screen composition
  (`app/(app)/dashboard/page.tsx`, replacing the current stub).

**Out of scope**
- **Health Score** and **AI Insights** — the current `/dashboard` stub's placeholder text
  mentions both, but per the specs index they belong to **SPEC-213** (`GET /health-score`,
  `GET /insights`), not this spec; SPEC-212 touches neither endpoint nor renders either surface.
  The design mockup (`docs/05-design/ds/pages/dashboard.html`) shows a combined page for
  illustration — SPEC-212 owns only its top portion (hero + metrics + allocation + sectors).
- Any write/mutation — this screen is 100% read-only, mirroring SPEC-103's backend scope.
- Wiring the shell's global "+ Adicionar ativo" button (SPEC-200's `TopBar`, currently a no-op
  on every screen) to open a holding-add modal — real, but a cross-screen decision bigger than
  this spec; see Open Questions.
- Any historical/time-series view (e.g. a balance chart over time) — the backend computes only
  the current snapshot; no history is persisted (SPEC-103 §15 Open Questions).

---

## 3. Functional Requirements

### FR-2121 — Portfolio Summary Hero (FR-004)

On entering `/dashboard`, fetch `GET /dashboard` and render the **current value** (the
investor's full patrimony/net worth) as the primary figure, with the **growth vs. cost basis**
(absolute centavos + relative bps) shown alongside it.

#### Acceptance Criteria

- [ ] The hero shows `current_value_centavos` as the headline figure, pt-BR formatted
      (`R$ 297.924,80`), in the numeric/tabular type role.
- [ ] Growth (`growth_centavos` + `growth_bps`) is shown as a signed figure — gain in
      `text-gain` with `▲`, loss in `text-loss` with `▼`, zero in neutral text with no arrow —
      labelled as growth **vs. cost basis** (what the backend actually computes:
      `current_value − total_invested`), never mislabelled as a monthly/period figure the
      backend does not track.
- [ ] Money/percentages are read directly from the response — **never summed, subtracted, or
      recomputed client-side** (BR-2121).

### FR-2122 — Key Metrics Row (FR-004)

Three at-a-glance figures alongside the hero: total invested (cost basis), monthly passive
income, and the growth figure again as its own labelled metric (mirrors the design mockup's
metric row) — all pt-BR formatted, all read from the same response.

#### Acceptance Criteria

- [ ] Shows `total_invested_centavos`, `monthly_income_centavos`, and the growth figure
      (same value as FR-2121, not recomputed) as three distinct metric cards.
- [ ] Each metric is legible on its own (a label + the pt-BR figure), consistent with the
      `metric-callout` component the design system already documents for this role.

### FR-2123 — Asset-Class Allocation (FR-005)

Shows the current-value share of each asset class (`fii`, `fixed_income`, `stocks`, `etfs`) from
`allocation[]`.

#### Acceptance Criteria

- [ ] Renders every entry in `allocation[]` as a spectrum-bar segment + legend item (reusing the
      existing `AllocationBar` component, SPEC-200) — label + pt-BR share (`formatShareBps`).
- [ ] A class with `share_bps: 0` (e.g. Stocks/ETFs, always 0 in the MVP per SPEC-103 D5) is
      **omitted from the legend** rather than shown as a confusing zero-width segment — the bar
      only ever shows classes the investor actually holds.
- [ ] If every class is 0 (empty portfolio), this section is not rendered standalone — it folds
      into the empty-portfolio state (FR-2126).

### FR-2124 — FII Sector Exposure (FR-005)

Shows the FII-only current-value share by sector (Logistics, Offices, Shopping, Hybrid, Paper,
Other) from `fii_sectors[]`, as its own breakdown distinct from FR-2123's asset-class view.

#### Acceptance Criteria

- [ ] Renders every entry in `fii_sectors[]` the same way as FR-2123 (spectrum bar + legend),
      with pt-BR sector labels (a label map, mirroring `lib/portfolio/labels.ts`'s pattern —
      the wire carries the backend's sector enum, the UI shows the pt-BR name).
- [ ] Shown only when the investor holds at least one FII (`allocation.fii.value_centavos > 0`);
      omitted entirely for a fixed-income-only portfolio, not shown empty.
- [ ] The `other` sector (unknown/no quote) is labelled distinctly ("Outros / sem cotação") so
      it reads as a data-quality signal, not a real sector choice (ties to FR-2125).

### FR-2125 — Stale-Quote Degradation Indicator

Surfaces `stale_tickers[]` (FIIs valued at cost basis because their latest quote is missing) as
a visible, honest signal — never hidden, never silently blended into the figures as if they were
fresh.

#### Acceptance Criteria

- [ ] When `stale_tickers` is non-empty, show a small inline notice near the hero/metrics (e.g.
      "HGLG11 avaliado pelo custo — cotação indisponível") listing the affected tickers.
- [ ] The notice is informational only (no retry action — the backend's own ingestion worker
      owns freshness, SPEC-006) and never blocks the rest of the dashboard from rendering.

### FR-2126 — Empty Portfolio State

An investor with no holdings at all (`GET /dashboard` returns a zeroed summary, `200`, per
SPEC-103 FR-1036) sees a clear empty state, not a dashboard full of confusing zeroes.

#### Acceptance Criteria

- [ ] `total_invested_centavos === 0 && current_value_centavos === 0` renders a dedicated empty
      state ("Sua carteira está vazia") with a CTA linking to `/portfolio` (Carteira, SPEC-211)
      — not the hero/metrics/allocation sections half-rendered with zeroes.
- [ ] This is distinct from a **loading** or **error** state (FR-2127) — a confirmed-empty `200`
      is not a failure.

### FR-2127 — Loading & Error States

Mirrors the established shell pattern (SPEC-210/SPEC-211): a loading skeleton while the request
is in flight, and a retry affordance on a transient failure.

#### Acceptance Criteria

- [ ] Loading → skeleton placeholders for the hero + metric row (no layout shift once data
      arrives).
- [ ] A transient error (network/5xx) → a clear message + a "Tentar novamente" retry button,
      never a blank page or an uncaught crash.

### FR-2128 — Money/Rates Stay Integer, Display-Only

No monetary or percentage value is ever computed, summed, or converted through a float on the
client — every figure is the backend's own integer centavos/bps, formatted to pt-BR only at the
render edge via the existing `lib/money.ts` helpers.

#### Acceptance Criteria

- [ ] `formatCentavos`/`formatShareBps` (already built, SPEC-200/211) are the only formatting
      path used; no new ad-hoc formatting, no `toFixed`/float division anywhere in this screen
      (the exact float-arithmetic mistake caught and fixed in SPEC-211's review — not repeated
      here).
- [ ] No hand-written DTOs — types come from `lib/api/schema.ts`'s generated `DashboardResponse`.

---

## 4. User Flows

### Main Flow

1. The authenticated user opens `/dashboard` (the shell's default landing route).
2. `GET /dashboard` fetches the computed summary + allocation + sectors.
3. Populated → hero, metric row, asset-class allocation, FII sector exposure, and any
   stale-ticker notice all render from the one response.
4. Empty portfolio → the empty state (FR-2126), pointing to Carteira.

### Alternative Flow — degraded data

1. A held FII has no stored quote; its current value falls back to cost basis server-side
   (SPEC-103 BR-1036) — the dashboard still renders fully, reconciled, with the ticker listed
   in the stale notice (FR-2125).

### Alternative Flow — transient failure

1. `GET /dashboard` fails (network/5xx) → the error state (FR-2127); retry re-fetches.

---

## 5. Business Rules

### BR-2121 — Every figure is read, never computed, client-side
The backend already guarantees reconciliation and determinism (SPEC-103 BR-1031/BR-1034); the
client's only job is display. No sum, percentage, or growth figure is ever derived from other
fields in the response — each is read as-is, mirroring SPEC-211's BR-2117 principle
("reference rates are read from the server, never computed client-side") applied here to the
*entire* screen, not just one field.

### BR-2122 — Money & rates are integers, display-only (one direction)
Unlike SPEC-211 (which also parses user input), this screen only ever **displays** — integer
centavos/bps in, pt-BR strings out, at the render edge, via `lib/money.ts`. No user-entered
money/rate exists on this screen (BR-2003/BR-2112's discipline, minus the input-parsing half).

### BR-2123 — No AI output, no money-output guards
This screen renders no LLM-generated content, so FR-013/FR-014 do not apply (mirrors SPEC-103's
own BR-1036 and SPEC-210/211's equivalent BRs). Its job is to make the *deterministic* facts
legible — the same facts SPEC-104's Fact Builder (backend) already reuses; SPEC-213 is where
those facts get an AI-generated explanation on the client.

### BR-2124 — Types from the generated contract
Request/response types come from `lib/api/schema.ts` (`DashboardResponse` and its nested
`summary`/`allocation`/`fii_sectors` shapes) — no hand-written DTOs (mirrors BR-2116).

### BR-2125 — Degradation is visible, not hidden
A stale-quote fallback (FR-2125) or a zero-share asset class (FR-2123) is a real state of the
investor's data, not an error to swallow silently or a success to overstate — the screen shows
it plainly rather than either hiding it or blocking on it.

### BR-2126 — Identity from the session
Screen is behind the `(app)` `RequireAuth` gate (SPEC-200); `GET /dashboard` carries the session
cookie; the client never sends or trusts a `user_id` (mirrors BR-2001/BR-2111).

---

## 6. Domain Model

Not applicable — the screen holds no domain model of its own. It renders the SPEC-103
`DashboardResponse` (summary + allocation + fii_sectors + stale_tickers), typed from the
generated contract. Local state is load state only (loading/error/data), no form state.

---

## 7. API Contract

**Consumes the existing SPEC-103 endpoint — declares none, changes none.** No `api/openapi.yaml`
edit belongs to this spec.

- `GET /dashboard` → `200 DashboardResponse` — `{summary: {total_invested_centavos,
  current_value_centavos, monthly_income_centavos, growth_centavos, growth_bps}, allocation:
  [{asset_class, value_centavos, share_bps}], fii_sectors: [{sector, value_centavos,
  share_bps}], stale_tickers: [string]}`. `401` if unauthenticated.

---

## 8. Data Model

Not applicable — no new tables, no client persistence. The response is cached ephemerally via
TanStack Query, matching SPEC-210/211's pattern; no mutation ever invalidates it from this
screen (nothing here writes), though a future cross-screen invalidation from Carteira's
create/update/delete could be a natural follow-up (see Post-Implementation Tasks — out of scope
here since SPEC-211 already shipped without it and this spec doesn't reopen SPEC-211).

---

## 9. Edge Cases

### Empty portfolio
`total_invested_centavos === 0 && current_value_centavos === 0` → the dedicated empty state
(FR-2126), not a zeroed dashboard.

### Every FII quote stale
All held FIIs valued at cost basis, all listed in `stale_tickers` — the dashboard still renders
(figures still reconcile, just conservatively), with the degradation notice prominent.

### Fixed-income-only portfolio
`allocation` shows only the `fixed_income` slice; `fii_sectors` is empty and the whole section
is omitted (FR-2124) rather than shown as an empty chart.

### All value in one asset class
Asset-class allocation legitimately renders a single 100%-width segment — not a bug, a real
state (mirrors SPEC-211's "prefixado never shows a reference" pattern: a degenerate-looking
display can be the correct one).

### Session expired mid-view
A `401` on `GET /dashboard` → the SPEC-200 auth handling clears state and routes to login.

### Transient failure loading
A network/5xx blip → the error + retry state (FR-2127); a successful retry replaces it cleanly.

---

## 10. Security Requirements

### Authentication
Screen is behind the `(app)` `RequireAuth` gate (SPEC-200); `GET /dashboard` carries the session
cookie.

### Authorization
The backend scopes the computation to the session `user_id` (SPEC-103 BR-1033); the client never
sends or trusts one.

### Data Protection
No secrets; portfolio figures are the user's own financial data — nothing logged to the console.

---

## 11. Observability

### Metrics / Logs / Traces
No new client instrumentation in the MVP; the backend already traces `GET /dashboard`
(SPEC-004/SPEC-103 FR-1038). Client error surfacing is UI-level (the shared error pattern).

---

## 12. Testing Strategy

### Unit / Component (Vitest + RTL)
- Loading/error/empty/populated rendering, each as its own state (FR-2126/FR-2127).
- Hero + metric row render the exact pt-BR figures for a representative `DashboardResponse`
  fixture, including a loss (red, `▼`) and a gain (green, `▲`) growth case.
- Asset-class allocation: a zero-share class is omitted from the legend; a single-class
  portfolio renders a full-width segment without error.
- FII sector exposure: omitted entirely for a fixed-income-only fixture; rendered correctly
  (incl. the `other` sector's distinct label) for a mixed fixture.
- Stale-ticker notice: absent when `stale_tickers` is empty; present and listing the right
  tickers when not.

### Integration
- Against a running backend: seed holdings (SPEC-211's own create flow) + market data, load
  `/dashboard`, assert the rendered figures match the backend's computed response.

### End-to-End (Playwright)
- A smoke test: register → add a holding (via Carteira) → visit `/dashboard` → see the hero
  reflect a non-zero patrimony. Gated to skip without a backend, mirroring SPEC-211's
  `e2e/portfolio.spec.ts`.

---

## 13. Definition of Done

- [ ] FR-2121…FR-2128 implemented; Epic 3 acceptance criteria satisfied.
- [ ] BR-2121…BR-2126 respected (read-only/no-client-computation, integer money display-only,
      no AI guards needed, generated types, visible degradation, identity from session).
- [ ] Consumes SPEC-103 only; **no `api/openapi.yaml` change**.
- [ ] Vitest/RTL + integration + gated E2E green in the `web/` CI gate.
- [ ] Reviewed by **frontend-reviewer** + **react-correctness-reviewer**.
- [ ] CHANGELOG updated; SPEC-212 + PLAN-212 flipped to Done; indexes updated.
- [ ] PT-BR lesson `docs/lessons/SPEC-212-aula.html` via **frontend-lesson-writer**.

---

## 14. Open Questions

1. **The shell's global "+ Adicionar ativo" button** (SPEC-200's `TopBar`, currently a no-op on
   every screen) — should visiting `/dashboard` wire it to open SPEC-211's add-holding modal, or
   leave it a Carteira-only affordance for now? Cross-screen coupling bigger than this spec;
   recommend leaving it a no-op here and deciding when/if a screen actually needs it — no
   screen has claimed it yet.
2. **Donut vs. spectrum bar for FII sector exposure** — the design mockup shows a donut chart
   for sectors and a spectrum bar for the hero's allocation preview (visually distinct), but the
   codebase only has the spectrum-bar `AllocationBar` component built (SPEC-200). Recommend
   reusing `AllocationBar` for both (FR-2123/FR-2124) rather than building a new donut component
   for this spec — a donut is a nice-to-have visual upgrade, not required by Epic 3's acceptance
   criteria, and building one is a PLAN-212-level decision, not a blocking one.
3. **Cross-screen cache invalidation** — editing a holding on Carteira (SPEC-211) doesn't
   currently invalidate the Dashboard's TanStack Query cache, so a user could see stale figures
   navigating Carteira → Dashboard without a manual refetch. Worth a small follow-up (a shared
   query key or an invalidation on holdings mutations) but not blocking for this spec's MVP —
   TanStack Query's default `refetchOnMount`/`staleTime` behavior already covers the common case
   reasonably well; revisit if it proves confusing in practice.
