# SPEC-213 — AI Insights, Rebalancing & Health Score Screens

## 1. Document Information

| Field        | Value                                                                 |
| ------------ | ---------------------------------------------------------------------- |
| Feature Name | AI Insights, Rebalancing & Health Score Screens                       |
| Feature ID   | SPEC-213                                                                |
| Version      | 0.1.0                                                                   |
| Status       | Draft                                                                   |
| Author       | Gabigol                                                                |
| Last Updated | 2026-07-14                                                             |
| Related PRD  | [Epic 5](../01-product/PRD.md) (FR-008–FR-010 insights), [Epic 6](../01-product/PRD.md) (FR-011 rebalancing), [Epic 7](../01-product/PRD.md) (FR-012 health score); **binding gates** FR-013 (explainability), FR-014 (non-advice), FR-019/020/021 |
| Consumes     | [SPEC-104](SPEC-104-ai-insight-engine.md) (`GET /insights`), [SPEC-105](SPEC-105-ai-rebalancing-assistant.md) (`POST /rebalancing`), [SPEC-106](SPEC-106-portfolio-health-score.md) (`GET /health-score`) over the [OpenAPI contract](../../api/openapi.yaml); built on [SPEC-200](SPEC-200-app-foundation.md); reuses `InsightCard`, `NonAdviceDisclaimer`, `AllocationBar` and the Aurora design system ([ADR-0006](../04-architecture/adr/ADR-0006-frontend-ui-stack-and-design-system.md)) |

---

## 2. Overview

### Purpose

Turn the `/insights` and `/health` stubs into the product's **AI reasoning surfaces** — the
frontend faces of SPEC-104 (explainable insights), SPEC-105 (contribution/rebalancing guidance),
and SPEC-106 (the Portfolio Health Score). This is the first frontend spec whose entire payload is
**LLM-generated, gated output**, so the two binding product guards — **explainability (FR-013)** and
**non-advice (FR-014)** — stop being a background note (as in SPEC-212's read-only dashboard) and
become the spec's defining, structural concern.

Concretely it delivers three surfaces:

- **`/insights` (Insights)** — the aggregated portfolio / allocation / market-context insights from
  `GET /insights`, **and** the interactive **"Assistente de Aporte"** (rebalancing) panel that posts
  a contribution amount to `POST /rebalancing` and shows suggested areas + their computed split +
  grounded named candidates.
- **`/health` (Saúde)** — the reproducible 0–100 Health Score gauge, its five-factor breakdown, and
  the optional gated AI narrative, from `GET /health-score`.

### Business Value

Everything the backend built through SPEC-104…106 exists to be *seen and trusted* here. Insights are
the payoff of "spot concentration, imbalance and risk without manual analysis" (Epic 5); the aporte
assistant is the investor's most actionable moment — "I have R$X this month, where do I focus?" (Epic
6); the Health Score is the single trustworthy signal Carla wants (Epic 7, Persona 2). The client's
job is to render that reasoning **honestly** — every AI card carries its explanation, every AI surface
carries the non-advice disclaimer, no figure is recomputed — so the product's core promise (a copilot,
never an advisor) is visible and safe *by construction*, not by reviewer vigilance.

### Scope

**In scope**

- **Insights list** (`GET /insights`) — every insight rendered through the existing `InsightCard`
  (explanation slot required by its prop contract), tagged by category (portfolio / allocation /
  market_context), with the non-advice disclaimer and degraded/empty states.
- **Rebalancing / "Assistente de Aporte"** (`POST /rebalancing`) — a contribution-amount input
  (pt-BR → integer centavos), the returned **areas** each with their **computed `suggested_share_bps`**
  (displayed verbatim, reusing `AllocationBar`) and `suggested_amount_centavos`, and the **named FII
  candidates** nested in the FII area — each a consideration, each explained, with the disclaimer.
  Embedded as a panel **on the Insights screen** (no new nav route — see Open Questions).
- **Health Score** (`GET /health-score`) — the 0–100 score as a gauge, the five-factor breakdown
  (each: sub-score, weight bps, computed explanation), and the optional AI narrative that degrades
  away without hiding the score.
- The data hooks (`lib/insights/*`, `lib/health/*`) and the screen compositions replacing the two
  stubs; the small presentational pieces these screens need (score gauge, factor row, area/candidate
  cards) built on Aurora.
- Loading, error, empty-portfolio, and LLM-degraded states for all three surfaces.

**Out of scope**

- **Projections** (`GET /projections`, SPEC-107) — its own screen, **SPEC-214**; this spec touches
  neither the endpoint nor the `/projections` stub.
- **The conversational copilot** (chat, streaming, the "tenho R$X" *chat* path) — **SPEC-215**. The
  aporte assistant here is a **structured form on the Insights screen**, not the chat widget; the two
  are distinct surfaces onto the same SPEC-105 backend (PRD §8 Epic 10 vs Epic 6).
- **Insight / rebalancing history** (FR-022 was deferred at SPEC-104 D4; conversation memory is
  SPEC-108/SPEC-215) — these screens are on-demand and stateless; no history view.
- Any new endpoint, any `api/openapi.yaml` change (a `SPEC-21x` screen consumes its twins over the
  frozen contract — root `web/CLAUDE.md`), and any change to the SPEC-200 nav (`/insights`, `/health`
  already exist; no `/rebalancing` route is added).
- The `include_asset_shares` per-candidate illustrative share — **opt-out for the MVP** (default
  `false`, the natural surface for it is the SPEC-215 chat "quanto em cada um?" turn, SPEC-105 D6).

---

## 3. Functional Requirements

### FR-2131 — Portfolio Insights List (FR-008/009/010, Epic 5)

On entering `/insights`, fetch `GET /insights` and render each returned insight as an `InsightCard`,
tagged by its category, with the human-readable **explanation always shown**.

#### Acceptance Criteria

- [ ] Every entry in `insights[]` renders as an `InsightCard` whose `explanation` prop is the
      response's `explanation` field — never omitted, never empty (the card's prop contract makes an
      explanation-less insight unrepresentable; FR-013 / `web/CLAUDE.md` structural guard).
- [ ] Insights are grouped/labelled by `category` (`portfolio` | `allocation` | `market_context`)
      using pt-BR labels (a label map mirroring `lib/portfolio/labels.ts`'s pattern — the wire carries
      the backend enum, the UI shows the pt-BR name).
- [ ] `title`/`detail` are the card body; no insight text is generated, transformed, or summarized on
      the client — it is displayed verbatim (facts/reasoning are the backend's; BR-2132).

### FR-2132 — Rebalancing "Assistente de Aporte" (FR-011, Epic 6)

An interactive panel on the Insights screen: the investor enters a contribution amount and the panel
posts `POST /rebalancing`, then renders the suggested **areas** with their **computed split** and
grounded candidates.

#### Acceptance Criteria

- [ ] A single money input accepts a pt-BR amount and is parsed to **integer centavos via
      `parseCentavos`** (never a float); an empty/malformed/≤0 amount is blocked at the edge with a
      clear message and **no request is sent** (mirrors SPEC-105 FR-1051's `400`, caught client-side
      first).
- [ ] On submit, `contribution_centavos` (int) is the **only** value sent; `include_asset_shares` is
      omitted/`false` (out of scope, FR-213 scope note). Identity is the session cookie, never a
      client `user_id` (BR-2136).
- [ ] Each returned area renders with its pt-BR class label, its **`suggested_share_bps` shown
      verbatim** (`formatShareBps`) and `suggested_amount_centavos` (`formatCentavos`), and its
      explanation — the split is **read, never recomputed or re-summed** client-side (BR-2131).
- [ ] The suggested split is visualized with the existing `AllocationBar` (areas as segments), so the
      "where to focus" reads at a glance, consistent with the dashboard's allocation view.
- [ ] The panel has its own loading (post in flight) and error/retry state, independent of the
      insights list on the same screen (one failing does not blank the other).

### FR-2133 — Named Candidate Assets as Considerations (FR-011, FR-019)

Within the FII area, render the grounded **named candidates** returned by the backend — each framed
explicitly as a *consideration for the user's own analysis*, never an order.

#### Acceptance Criteria

- [ ] Each entry in an area's `candidates[]` renders its `ticker`, pt-BR `sector` label, `title`/
      `detail`, and its **required `explanation`** (reusing the `InsightCard` explanation-slot
      discipline, or an equivalent card that also makes the explanation non-optional).
- [ ] Candidates are visibly framed as considerations (e.g. a "para você analisar" affordance), never
      as an imperative buy — no "compre", no quantity, no price target rendered by the client
      (FR-014); the client displays only what the gated backend returned and adds no directive of its
      own.
- [ ] `illustrative_share_bps` is **not** requested nor rendered in the MVP (default off); if a future
      response carries it, the client still treats it as a consideration only (forward-safe).

### FR-2134 — Portfolio Health Score Gauge + Factor Breakdown (FR-012, Epic 7)

On entering `/health`, fetch `GET /health-score` and render the 0–100 score prominently with its
five-factor breakdown.

#### Acceptance Criteria

- [ ] The integer `score` (0–100) is shown as the headline (a gauge/dial or ring), in the
      numeric/tabular type role — displayed verbatim, never derived from the factors client-side
      (BR-2131).
- [ ] Every entry in `factors[]` renders with its pt-BR factor label (`diversification` |
      `concentration` | `liquidity` | `goal_alignment` | `risk_exposure`), its `score` (0–100), its
      `weight_bps` (`formatShareBps`), and its computed `explanation` (always shown; FR-013).
- [ ] The breakdown is the **explanation of record** (SPEC-106 BR-1062) — it is always present even
      when the AI narrative is absent (FR-2135); the screen never hides a factor.

### FR-2135 — Optional AI Health Narrative (FR-013, degradable)

Render the gated AI narrative when present, and degrade it away cleanly when not — the score and
breakdown always stand.

#### Acceptance Criteria

- [ ] When `narrative_available` is `true` and `narrative` is non-empty, it renders as a distinct,
      clearly-labelled "análise do copiloto" block, accompanied by the non-advice disclaimer (FR-2137).
- [ ] When `narrative_available` is `false`, the narrative block is omitted (a small "análise
      indisponível no momento" note is acceptable) and the **score + factor breakdown still render in
      full** — a narrative outage is not a screen failure (SPEC-106 FR-1063).

### FR-2136 — Explainability Slot Is Structural (FR-013)

The explanation is not an optional decoration on any AI surface in this spec — it is enforced by the
component prop contract, mirroring the backend's `Gated` `Insighter` (`web/CLAUDE.md`).

#### Acceptance Criteria

- [ ] Every AI card (insight, area, candidate, factor) is rendered by a component that makes its
      `explanation` a **required, non-empty prop** — an explanation-less item is unrepresentable, not
      merely un-styled (`InsightCard` already enforces this; new cards follow the same contract).
- [ ] No client code constructs, paraphrases, or synthesizes AI-facing reasoning text — it only
      displays the backend's gated `explanation`/`detail`/`narrative` verbatim.

### FR-2137 — Non-Advice Disclaimer Is Non-Optional (FR-014)

Every surface that renders AI output carries the `NonAdviceDisclaimer`; the render path cannot omit it.

#### Acceptance Criteria

- [ ] The Insights screen (insights list **and** the aporte panel when it shows results) and the
      Health screen (whenever the narrative renders, and on the screen generally as it carries AI
      reasoning) each render `NonAdviceDisclaimer`.
- [ ] The client never renders a buy/sell order, a ticker-to-buy imperative, a transaction quantity,
      or a price/entry-exit target of its own — it surfaces only areas/considerations + the disclaimer
      (root `web/CLAUDE.md`, PRD §6.8).

### FR-2138 — Graceful Degradation, Empty & Error States (per surface)

Each of the three surfaces handles the backend's degraded/empty contracts and transient failures
independently and honestly.

#### Acceptance Criteria

- [ ] **Insights `available:false`** (full LLM outage → `200`, empty `insights[]`, SPEC-104 D5) → a
      clear "insights temporariamente indisponíveis" state, **not** an error and **not** an empty void;
      the rest of the app stays usable.
- [ ] **Insights `available:true` + empty `insights[]`** (empty portfolio, SPEC-104 FR-1047) → a
      friendly "adicione ativos para receber insights" empty state with a CTA to `/portfolio`.
- [ ] **Rebalancing `available:false`** → the aporte panel shows "assistente temporariamente
      indisponível", the entered amount preserved; a valid degraded `200` is not treated as an error.
- [ ] **Health**: an empty portfolio returns `score: 0` + an "adicione ativos" explanation (SPEC-106
      FR-1066) — rendered as a defined empty-ish state, not a misleading "0/100 = terrible portfolio".
- [ ] A transient network/5xx on any of the three fetches → that surface's own error + "Tentar
      novamente" retry (the SPEC-210/211/212 pattern), never a blank page or uncaught crash; a `401`
      routes to login via the SPEC-200 auth handling.

### FR-2139 — Money, Rates & Scores Stay Integer, Display-Only (BR-1022)

No monetary, percentage, or score value is computed, summed, or floated on the client — every figure
is the backend's own integer centavos / bps / score, formatted to pt-BR only at the render edge.

#### Acceptance Criteria

- [ ] `formatCentavos` / `formatShareBps` / `formatBps` (SPEC-200/211) are the only display path;
      `parseCentavos` is the only input path (the aporte amount). No `toFixed`, no float division, no
      ad-hoc formatting anywhere (the SPEC-211 float mistake is not repeated).
- [ ] The computed rebalancing split (`suggested_share_bps`) and the health `score`/`weight_bps` are
      displayed as received — the client asserts nothing about them summing to 10 000 / 100 (that is
      the backend's reconciliation invariant, SPEC-105 FR-1053a / SPEC-106 FR-1062), it just renders.

### FR-2140 — Types From the Generated Contract Only

Request/response types come from `lib/api/schema.ts` — no hand-written DTOs.

#### Acceptance Criteria

- [ ] `InsightsResponse`, `RebalancingResponse` (+ request body), and `HealthScoreResponse` are used
      from the generated `components["schemas"]` (mirrors BR-2124); no re-declared shapes.
- [ ] The three hooks use the typed `openapi-fetch` client (`api.GET`/`api.POST`) over TanStack
      Query, matching `lib/dashboard/dashboard.ts`'s shape.

---

## 4. User Flows

### Main Flow — Insights

1. The authenticated user opens `/insights`.
2. `GET /insights` fetches the aggregated, gated insights.
3. Populated → each insight renders as an explained `InsightCard`, grouped by category, with the
   non-advice disclaimer.

### Main Flow — Aporte (rebalancing)

1. On `/insights`, the user enters a contribution amount in the "Assistente de Aporte" panel.
2. The amount is parsed to integer centavos and posted to `POST /rebalancing`.
3. The panel renders the suggested areas (computed split via `AllocationBar` + per-area explanation)
   and the grounded FII candidates nested in the FII area — each a consideration, with the disclaimer.

### Main Flow — Health

1. The user opens `/health`.
2. `GET /health-score` returns the computed score + factor breakdown (+ optional narrative).
3. The gauge shows the 0–100 score; the five factors render with sub-score, weight, and explanation;
   the narrative renders when available.

### Alternative Flow — LLM degraded

1. `available:false` (insights or rebalancing) or `narrative_available:false` (health) → the affected
   surface shows its "temporariamente indisponível" state; the score/breakdown and the rest of the app
   remain usable.

### Alternative Flow — empty portfolio

1. Insights → friendly "adicione ativos" empty state → CTA to `/portfolio`.
2. Health → `score: 0` + "adicione ativos" explanation, shown as a defined empty-ish state.

### Alternative Flow — invalid aporte / transient failure

1. Empty/≤0/malformed amount → blocked at the edge, no request sent.
2. Network/5xx on any fetch → that surface's error + retry.

---

## 5. Business Rules

### BR-2131 — Every figure is read, never computed, client-side
The backend guarantees reconciliation and determinism (SPEC-104/105/106); the client's only job is
display. No score, share, split, or centavos figure is derived from other fields — each is read as-is
(the SPEC-212 BR-2121 principle, extended to three AI surfaces). The split's Σ = 10 000 and the
score's factor-weighted mean are backend invariants the client renders, never re-verifies.

### BR-2132 — AI text is displayed verbatim, never generated on the client
Insight titles/details/explanations, area/candidate reasoning, and the health narrative are the
backend's **gated** output. The client never constructs, paraphrases, summarizes, or augments
AI-facing reasoning — doing so would bypass the FR-013/FR-014 gates the backend owns. The client adds
only pt-BR *labels* (category/sector/factor names), not reasoning.

### BR-2133 — The binding guards are structural, not per-screen discipline
Explainability (FR-013) is enforced by required, non-empty `explanation` props (`InsightCard` and its
siblings); the non-advice disclaimer (FR-014) is a non-optional element on every AI surface. These
mirror the backend's `Gated` `Insighter` decorator — enforced by construction (`web/CLAUDE.md`), not
by reviewer vigilance. A card with no explanation, or an AI surface with no disclaimer, must fail to
compile/render, not merely look wrong.

### BR-2134 — Money & rates are integers; only the aporte amount is user input
Every displayed figure is integer centavos/bps/score → pt-BR at the render edge (`lib/money.ts`). The
single user-entered value on these screens is the **aporte amount**, parsed to integer centavos via
`parseCentavos` (never a float) before it reaches the request body (SPEC-211's BR-2112 discipline).

### BR-2135 — Types from the generated contract
Request/response types come from `lib/api/schema.ts` (`InsightsResponse`, `RebalancingResponse`,
`HealthScoreResponse`) — no hand-written DTOs (mirrors BR-2124); this spec declares no endpoint and
carries no `api/openapi.yaml` change.

### BR-2136 — Identity from the session
All three screens are behind the `(app)` `RequireAuth` gate (SPEC-200); each request carries the
session cookie; the client never sends or trusts a `user_id` (mirrors BR-2001/BR-2111/BR-2126).

### BR-2137 — Degradation is visible, not hidden
An `available:false` (insights/rebalancing) or `narrative_available:false` (health) is a real state of
the AI layer, not an error to swallow nor a success to fake. The screen shows it plainly and keeps the
deterministic parts (health score + breakdown) intact — never blocking the whole view on the LLM.

---

## 6. Domain Model

Not applicable — the screens hold no domain model of their own. They render the SPEC-104
`InsightsResponse`, the SPEC-105 `RebalancingResponse`, and the SPEC-106 `HealthScoreResponse`, all
typed from the generated contract. Local state is: per-surface load state (loading/error/data) and the
aporte panel's single controlled input + its own request state. No client-side domain computation.

---

## 7. API Contract

**Consumes the three existing backend endpoints — declares none, changes none.** No `api/openapi.yaml`
edit belongs to this spec (`SPEC-21x` rule).

- `GET /insights` → `200 InsightsResponse` — `{insights: [{category, title, detail, explanation}],
  disclaimer, available}`. `401` if unauthenticated. (SPEC-104 FR-1048.)
- `POST /rebalancing` — body `{contribution_centavos:int64(>0), include_asset_shares?:bool}` →
  `200 RebalancingResponse` — `{areas: [{class, suggested_share_bps, suggested_amount_centavos, title,
  detail, explanation, candidates: [{ticker, sector, title, detail, explanation,
  illustrative_share_bps?}]}], disclaimer, available}`. `400` on a bad amount, `401` if unauthenticated.
  (SPEC-105 FR-1056.)
- `GET /health-score` → `200 HealthScoreResponse` — `{score:int(0–100), factors: [{name, score,
  weight_bps, explanation}], narrative, narrative_available, disclaimer}`. `401` if unauthenticated.
  (SPEC-106 FR-1065.)

---

## 8. Data Model

Not applicable — no new tables, no client persistence. Responses are cached ephemerally via TanStack
Query (matching `lib/dashboard/dashboard.ts`). The rebalancing result is a `POST` mutation result held
in local/mutation state, not a cached query keyed by user (its input is the transient aporte amount).
No mutation on these screens invalidates any other query (they are read/compute-only aside from the
aporte `POST`, which persists nothing — SPEC-105 has no persisted entity).

---

## 9. Edge Cases

### Empty portfolio
- Insights → `available:true`, empty `insights[]` → the "adicione ativos" empty state + CTA to
  `/portfolio` (not a degraded state).
- Health → `score: 0` + "adicione ativos" explanation → a defined empty-ish state, not "0 = unhealthy".
- Aporte → SPEC-105 still returns first-investment area guidance for a contribution even on an empty
  portfolio (FR-1053) — the panel renders it normally.

### LLM fully unavailable
- Insights / rebalancing → `available:false` → the surface's "temporariamente indisponível" state.
- Health → `narrative_available:false` → narrative omitted; **score + breakdown still render** (the
  deterministic core is unaffected).

### Invalid aporte amount
`≤ 0`, empty, or malformed → blocked at the edge with a message; no request sent (the client mirror of
SPEC-105's `400`).

### Grounded candidates only
The backend's grounding guard has already dropped any unknown/hallucinated ticker (SPEC-105 BR-1053);
the client renders whatever candidates it receives without second-guessing — but also never invents or
appends one of its own.

### Narrative present but insights down (or vice-versa)
The three fetches are independent — the health narrative can render while `GET /insights` is degraded,
and the insights list can render while the aporte panel is mid-request or errored. One surface's state
never dictates another's.

### Session expired mid-view
A `401` on any fetch → the SPEC-200 auth handling clears state and routes to login.

### Long AI text
Insight/narrative/explanation text is display-only and may be long — it wraps and never overflows its
card; nothing is truncated in a way that hides the explanation (FR-013 must remain fully readable).

---

## 10. Security Requirements

### Authentication
All three screens are behind the `(app)` `RequireAuth` gate (SPEC-200); each request carries the
session cookie.

### Authorization
The backend scopes every computation to the session `user_id` (SPEC-104/105/106 BR-*43); the client
never sends or trusts one. The aporte amount is the only client-supplied value and is validated `> 0`
at the edge (defence in depth; the backend re-validates).

### Data Protection
No secrets on the client; AI output and portfolio figures are the user's own data — never logged to the
console. The non-advice guard is a **product-safety control**: the client renders only gated backend
output and adds no directive, so it cannot manufacture advice the backend would have blocked.

---

## 11. Observability

### Metrics / Logs / Traces
No new client instrumentation in the MVP; the backend already traces `GET /insights`, `POST
/rebalancing`, and `GET /health-score` and records the Insighter's AI spans (no prompt/PII — SPEC-005
BR-505). Client error surfacing is UI-level (the shared per-surface error pattern). The client logs no
AI text, facts, or the aporte amount.

---

## 12. Testing Strategy

### Unit / Component (Vitest + RTL)
- **Insights**: populated list renders one explained `InsightCard` per item, grouped by pt-BR category
  label; `available:false` → the degraded state; `available:true` + empty → the empty state + CTA;
  transient error → retry.
- **Guard structure**: an insight/area/candidate/factor fixture with a missing/empty `explanation` is
  a **type error / render failure**, not a silently blank card (the FR-2136 contract); the
  `NonAdviceDisclaimer` is present on every AI surface that renders output (FR-2137).
- **Aporte panel**: a valid pt-BR amount parses to the exact integer centavos and is the only field
  posted; `≤0`/empty/malformed is blocked with no request; a `RebalancingResponse` fixture renders the
  areas with `suggested_share_bps` verbatim (`formatShareBps`), the `AllocationBar`, and nested
  candidates framed as considerations; `available:false` → the panel's degraded state with the amount
  preserved.
- **Health**: score gauge shows the integer score verbatim; every factor renders sub-score + weight +
  explanation; `narrative_available:true` renders the narrative + disclaimer, `false` omits it while
  the score + breakdown stand; `score:0` empty-portfolio fixture renders the defined empty-ish state.
- **Money discipline**: no `toFixed`/float path anywhere; `parseCentavos`/`formatCentavos`/
  `formatShareBps`/`formatBps` are the only money/rate paths; types come from `schema.ts` (no hand DTOs).

### Integration
- Against a running backend with the **fake Insighter** (SPEC-104/105/106 integration posture): seed
  holdings + quotes + profile + macro, load `/insights` and `/health` and submit an aporte, assert
  every rendered insight/area/candidate/factor carries an explanation and the disclaimer is present
  (the gates hold end-to-end on the client), and that a forced LLM outage degrades each surface as
  specified without breaking the health score.

### End-to-End (Playwright)
- A smoke test: register → add a holding → visit `/insights` (see at least the disclaimer + a
  populated-or-degraded state, no crash), submit an aporte amount (see areas or the degraded state),
  visit `/health` (see the 0–100 gauge + factor breakdown). Gated to skip cleanly without a backend,
  mirroring `e2e/dashboard.spec.ts`.

---

## 13. Definition of Done

- [ ] FR-2131…FR-2140 implemented; Epic 5/6/7 acceptance criteria (the client half) satisfied.
- [ ] BR-2131…BR-2137 respected (read-only/no-client-computation, AI text verbatim, **guards
      structural**, integer money incl. the aporte input, generated types, identity from session,
      visible degradation).
- [ ] Consumes SPEC-104/105/106 only; **no `api/openapi.yaml` change**, **no SPEC-200 nav change**.
- [ ] Vitest/RTL + integration + gated E2E green in the `web/` CI gate; typecheck + lint + build clean.
- [ ] Reviewed by **frontend-reviewer** + **react-correctness-reviewer** (explicitly on the FR-013/
      FR-014 client guards and money-as-integer).
- [ ] Suggest `/security-review` (this spec renders AI output — the non-advice surface is a safety
      control).
- [ ] CHANGELOG updated; SPEC-213 + PLAN-213 flipped to Done; indexes updated.
- [ ] PT-BR lesson `docs/lessons/SPEC-213-aula.html` via **frontend-lesson-writer** (product-focused:
      what the insights/aporte/health surfaces deliver and how the guards make explainability &
      non-advice tangible on screen).

---

## 14. Open Questions

1. **Where the aporte (rebalancing) lives.** The SPEC-200 nav is frozen and has no `/rebalancing`
   route (chat, the other "tenho R$X" surface, is deliberately a floating widget, not a route). This
   spec **embeds the aporte assistant as a panel on `/insights`** — the lowest-friction home that adds
   no nav route and keeps rebalancing next to the insights it complements. Alternative: a dedicated
   `/rebalancing` route (a nav change, reopening SPEC-200). **Recommend the embedded panel**; revisit a
   dedicated route only if the panel proves cramped alongside the insights list.
2. **Score gauge: build vs. reuse.** SPEC-106's score wants a 0–100 gauge/dial; the codebase has
   `AllocationBar` (a spectrum bar) but no radial gauge. **Recommend** a small, self-contained gauge
   component (a semicircle/ring in Aurora tokens) built here rather than pulling in a chart dependency
   — Recharts (ADR-0006's chosen chart lib) is reserved for the time-series work in SPEC-214; a single
   scalar gauge does not justify it. A PLAN-213 decision, not blocking.
3. **`include_asset_shares` (per-candidate illustrative %).** Left **off** for the MVP (SPEC-105 D6:
   its natural surface is the SPEC-215 chat "quanto em cada um?" turn). Worth revisiting once chat
   ships, if users want the per-candidate split on the structured screen too — additive, non-blocking.
4. **Cross-screen freshness.** Like SPEC-212, editing a holding on Carteira doesn't invalidate these
   AI queries; TanStack Query's `staleTime`/`refetchOnMount` covers the common case. A shared
   invalidation on holdings mutations is a small follow-up (shared with SPEC-212's open question), not
   blocking here.
