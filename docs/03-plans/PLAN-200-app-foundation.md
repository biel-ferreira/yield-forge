# PLAN-200 — Frontend App Foundation

## 1. Document Information

| Field           | Value                                   |
| --------------- | --------------------------------------- |
| Plan Name       | Frontend App Foundation                 |
| Related Feature | Frontend App Foundation                 |
| Related Spec    | [SPEC-200](../02-specs/SPEC-200-app-foundation.md) (Approved) |
| Version         | 0.1.0 (Draft)                           |
| Status          | Draft                                   |
| Author          | Gabigol                                 |
| Last Updated    | 2026-07-01                              |

> **Phase-order note.** The standard template phase order (Domain → Persistence →
> Application → API → Observability → Testing → Documentation) is backend-shaped. This is
> a **frontend** foundation spec, so each phase is mapped to its frontend analogue (scaffold
> → tokens/components → data client → auth → shell → quality → testing → docs). The
> principle is preserved: every phase leaves the build green and is independently reviewable.

---

## 2. Objective

### Goal

Stand up the Next.js web client under `web/` and everything every `SPEC-21x` screen depends
on: the app shell (sidebar + top bar + **global copilot overlay slot**), the **Aurora design
tokens as code**, the **typed API client generated from `api/openapi.yaml`**, authenticated
session handling against SPEC-003, and the pt-BR money/rate formatting edge — deployed to the
Vercel free tier. No product feature screen ships here; this is the frontend's `SPEC-001`.

### Expected Outcome

A running, deployable Next.js + TypeScript app where a user can log in, stay authenticated
across reloads, and land on an (empty-state) shell with the design system applied and the
copilot launcher mounted — with the binding guards (`InsightCard` explanation, non-advice
disclaimer) and money-as-integer conventions **structural on the client**, and a drift check
keeping the generated API types in lockstep with the contract.

---

## 3. Scope

### Included

- `web/` Next.js (App Router) + React + TypeScript (`strict`) scaffold, isolated from the Go module.
- Aurora **tokens as code** (`tokens.css` custom properties + Tailwind theme) generated from
  `docs/05-design/design-system.md`, plus the guard components + core primitives ported to React.
- **Typed API client** from `api/openapi.yaml` (`openapi-typescript` + `openapi-fetch`), TanStack
  Query, a **type-drift check**, the money/rate **formatting helper**, and the **SSE streaming
  transport** (so SPEC-215 chat is not a rework).
- **Auth & session** against SPEC-003 (`/auth/login`, `/auth/logout`, `/auth/me`), protected-route
  gating, and the cross-origin cookie posture.
- **App shell**: sidebar (Painel · Carteira · Insights · Saúde · Projeções · Perfil) + top bar +
  content area + **global overlay slot** hosting the floating copilot launcher (mount point + shell
  open/closed state only), shared loading/error boundaries, `{"error":...}` envelope handling, responsive.
- Quality gates (bundle budget, money-no-float lint), the test suite, Vercel deploy, and the SDD closeout.

### Excluded

- Any `SPEC-21x` **feature screen** content (Profile form, Portfolio CRUD, Dashboard data, Insights,
  Health, Projections) — those are their own specs. SPEC-200 ships **stubbed routes** only.
- The **copilot widget implementation** (bubbles/streaming UI/turn logic) — that is **SPEC-215**;
  SPEC-200 owns only the mount slot + open/closed shell state.
- Any backend change: **no new endpoints, no `api/openapi.yaml` edit** (this spec adds none).
- Client-side error tracking/tracing service (deferred; rely on backend OTel, SPEC-004).

---

## 4. Dependencies

### Technical Dependencies

- **`api/openapi.yaml`** — the frozen contract the typed client is generated from (source of truth).
- **SPEC-003** auth (`/auth/login`, `/auth/logout`, `/auth/me`, the HttpOnly session cookie).
- **`docs/05-design/design-system.md`** — the Aurora token source (colors/gradients/glow/type/spacing/components).
- **ADR-0004** (mono-repo `web/`, path-scoped CI, OpenAPI contract) and **ADR-0006** (UI stack: Next.js +
  TS + Tailwind + shadcn/ui, `openapi-typescript`/`openapi-fetch` + TanStack Query, Recharts, Claude Design).

### External Dependencies

- Node ≥ 20 LTS + a package manager (**pnpm** recommended: fast, disk-efficient; npm acceptable).
- Free tiers only (ADR-0003): **Vercel** (host), **Google Fonts / self-host** for Inter · Fraunces ·
  IBM Plex Mono (all SIL OFL), all OSS libraries.

### Blocking Decisions

| # | Decision | Resolution (this plan) |
|---|----------|------------------------|
| D1 | **Cross-origin cookie/CORS** for the SPEC-003 HttpOnly cookie | **Same-origin via a Next.js rewrite/proxy**: the browser calls `/{api}/…` on the web origin, Next proxies to the backend. The `SameSite=Lax` HttpOnly cookie then works **unchanged** — no CORS, no `SameSite=None`. (Fallback if proxy is undesirable: CORS `credentials` + `SameSite=None; Secure`.) |
| D2 | **SSR vs CSR per route** | **CSR by default** for authenticated data (client fetch via TanStack Query); SSR reserved for future public/SEO routes. Keeps auth simple (cookie read client-side via `/auth/me`). |
| D3 | **Test stack** | **Vitest + React Testing Library** (unit/component) + **Playwright** (smoke E2E). No backend test framework applies. |
| D4 | **Auth UI ownership** | Login/register live **inside SPEC-200** (FR-2003) for the MVP; not split to a separate `SPEC-201` unless it grows. |
| D5 | **SSE streaming now?** | **Yes** — build the streaming transport (fetch + `ReadableStream`) in Phase 3 so SPEC-215 chat reuses it. Confirm the backend chat streaming shape (SPEC-108) before finalizing the client contract. |

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| Repository root | Add top-level `web/` (own `package.json`, lockfile, tooling); `.gitignore` for `node_modules/` + build output (ADR-0004) |
| CI | Add **path-scoped** `web/**` pipeline (build/lint/typecheck/test + the OpenAPI type-drift check); Go build untouched |
| `api/openapi.yaml` | **Read-only** — consumed to generate types; **not modified** |
| `docs/05-design/design-system.md` | **Read-only** — token source; may add a `copilot-fab`/`copilot-panel` note (design follow-up, not code) |

### New Components

| Component | Purpose |
| --------- | ------- |
| `web/` Next.js app | The web client shell |
| `web/…/styles/tokens.css` + Tailwind theme | Aurora tokens as code (single source both previews and React consume) |
| `web/…/lib/api/` (generated types + `openapi-fetch` client + query hooks) | Typed API access from `api/openapi.yaml` |
| `web/…/lib/money.ts` | `formatCentavos` / `formatBps` (pt-BR) — the money-format edge |
| `web/…/lib/stream.ts` | SSE/streaming transport for chat (SPEC-215 reuse) |
| `web/…/auth/` (session provider, guards, login/logout) | Session against SPEC-003 |
| `web/…/components/ui/` | Ported design-system components (guards + primitives) |
| `web/…/app/` shell (layout, sidebar, top bar, copilot slot, boundaries) | The app shell |

---

## 6. Implementation Strategy

### Approach

**Incremental, phase-by-phase**, each leaving `web/` building/linting/type-checking green and
independently reviewable. Order is chosen so the app is runnable early and de-risks the two hardest
seams first: the **typed contract** (Phase 3) and **auth across origins** (Phase 4). Feature routes
are **stubbed** placeholders; no product data.

### Rollout Method

**Incremental.** SPEC-200 is not user-facing on its own; it merges behind the existing app (no public
launch). First real users arrive with the `SPEC-21x` screens.

### Rollback Strategy

`web/` is fully isolated (ADR-0004): reverting the `web/**` changes / disabling the Vercel deploy
removes the frontend with **zero** impact on the Go backend. No data migrations, no shared runtime.

---

## 7. Implementation Phases

### Phase 1 — Project Foundation & Toolchain  *(≈ scaffold)*

#### Tasks
- [ ] Create `web/` Next.js (App Router) + React + **TypeScript `strict`**; own `package.json` + lockfile.
- [ ] Tailwind CSS + base config; ESLint + Prettier; `.gitignore` for `node_modules/` + `.next/` (ADR-0004).
- [ ] Path-scoped CI for `web/**` (install, typecheck, lint, build); confirm the Go pipeline is untouched.
- [ ] `.env` contract for the client (API base URL) + `.env.example`; **no secrets in the bundle**.

#### Deliverables
- `web/` builds and lints clean in isolation; CI green on a trivial page.

---

### Phase 2 — Design Tokens & Core Components  *(≈ domain/model: the UI vocabulary)*

#### Tasks
- [ ] Generate **`tokens.css`** (CSS custom properties) + a **Tailwind theme** extension from
      `design-system.md` (colors, gradients, glow, radii, spacing, type roles); wire the three fonts.
- [ ] Port the **guard components** to React with contracts that make violations unrepresentable:
      `InsightCard` (**required** `explanation` prop — FR-013) and `NonAdviceDisclaimer` (FR-014).
- [ ] Port core primitives (Button incl. the glowing-outline variant, Card/glass, Input, Badge,
      spectrum AllocationBar) via Tailwind + shadcn/ui (copy-in); **no** Buy/Sell/order component exists.
- [ ] Dark-first theming; a `data-theme` light fallback that drops the ambient glow.

#### Deliverables
- A tokened component library; TSDoc cites the governing spec (e.g. `(SPEC-200 FR-2004)`).

---

### Phase 3 — Typed API Client & Data Layer  *(≈ persistence/data access)*

#### Tasks
- [ ] `openapi-typescript` script generating types from `api/openapi.yaml`; commit generated output.
- [ ] **Type-drift check** in CI: fails if committed types differ from a fresh generation (client mirror
      of the backend `openapi_test.go`).
- [ ] `openapi-fetch` typed client + **TanStack Query** setup (caching, invalidation, `401` handling hook).
- [ ] **`money.ts`**: `formatCentavos(int) → "R$ 1.234,56"`, `formatBps(int) → "10,50%"` (pt-BR, exact);
      money/rate fields typed and carried as **integers**, formatted only at the render edge (BR-2003).
- [ ] **`stream.ts`**: SSE/`ReadableStream` transport for chat token streaming (D5; SPEC-215 reuse).

#### Deliverables
- Typed client callable end-to-end against a running backend; drift check green; formatter table-tested.

---

### Phase 4 — Auth & Session  *(≈ application)*

#### Tasks
- [ ] Implement D1 **same-origin proxy** (Next rewrites) so the SPEC-003 HttpOnly `SameSite=Lax` cookie works.
- [ ] Session provider: `/auth/login` → `/auth/me` resolves identity (**server is the authority — never
      trust client-stored user id**, BR-2001); `/auth/logout` clears it.
- [ ] **Protected-route gate**: unauthenticated access redirects to login; a `401` on any call clears state
      and redirects **without** rendering protected data (FR-2003, edge cases).
- [ ] Login (and register, D4) UI using the design system.

#### Deliverables
- Full login → reload-persist → logout flow works against a local backend; `401` handling verified.

---

### Phase 5 — App Shell, Routing & Copilot Slot  *(≈ API/edge: the user-facing surface)*

#### Tasks
- [ ] Authenticated layout: **248px sidebar** (Painel · Carteira · Insights · Saúde · Projeções · Perfil,
      active-item gold edge) + top bar (greeting, theme toggle, primary action) + content area, per the
      Aurora **Dashboard mockup** (`docs/05-design/ds/pages/dashboard.html`).
- [ ] **Global overlay slot** mounting the floating copilot launcher on every authenticated screen — mount
      point + open/closed shell state **only** (widget internals are SPEC-215).
- [ ] **Stubbed** feature routes (empty states) for each nav item; shared **loading + error boundaries**;
      `{"error":"..."}` envelope surfaced via a shared pattern.
- [ ] Responsive: sidebar → bottom tabs on mobile; reduced glow on small/low-end devices.

#### Deliverables
- A navigable, responsive, authenticated shell with the copilot launcher mounted and empty-state routes.

---

### Phase 6 — Quality Gates & Observability  *(≈ observability)*

#### Tasks
- [ ] **Frontend review lens (the primary quality gate):** the review-before-closing runs the
      **frontend-reviewer** + **react-correctness-reviewer** subagents — the Go reviewers don't apply
      to React/TS. `/spec-implement` and `/pr-review` are made **track-aware** so a `web/` change
      selects these. (Under `.claude/agents` + `.claude/commands`.)
- [ ] **Format + reminder hooks:** `prettier-edited` (PostToolUse — Prettier-on-edit, the mirror of
      `gofmt-edited`; skips the generated `schema.ts`) and `on-stop-web` (Stop — reminder-only: run the
      `web/` gate, update CHANGELOG, regen types). Registered in `.claude/settings.json`.
- [ ] **Money-no-float** enforced by the `frontend-reviewer` lens + the single `money.ts` render edge
      (a bespoke ESLint rule is deferred by the rule-of-three — add it only if it bites).
- [ ] **Bundle-size budget** tracked in CI (informational); reduced-motion / reduced-glow fallback validated.
- [ ] Accessibility pass: WCAG AA contrast over the dark canvas (gold-on-dark, muted text); colorblind-safe
      gain/loss (arrow + text, never color alone).
- [ ] Confirm **no client tracing** in MVP (deferred); backend OTel (SPEC-004) covers the API calls made.

#### Deliverables
- The frontend review agents + format/reminder hooks are in place (the `web/` quality gate, replacing
  the Go gate); CI tracks the bundle budget; a documented a11y check passes.

---

### Phase 7 — Testing

#### Unit / Component Tests (Vitest + React Testing Library)
- [ ] `money.ts` formatting (table-driven, exact pt-BR output).
- [ ] OpenAPI **type-drift** check (generated types match the committed spec).
- [ ] Guard-component contracts: `InsightCard` requires `explanation`; `NonAdviceDisclaimer` presence.

#### Integration Tests
- [ ] Auth flow against a running backend: login → `/auth/me` → protected route → logout; `401` redirect.

#### End-to-End Tests (Playwright)
- [ ] Smoke: unauthenticated redirect → login → authenticated shell renders (with copilot launcher) → logout.
      Gated to skip cleanly without a running backend.

#### Deliverables
- Green unit/component + auth integration + smoke E2E suites in CI.

---

### Phase 8 — Documentation & Closeout

#### Tasks
- [ ] `web/README.md` (run, build, generate-types, test, deploy) + root README pointer.
- [ ] **CHANGELOG** `[Unreleased]` entry (same change).
- [ ] **No `api/openapi.yaml` change** — assert it (this spec adds no endpoint); note it explicitly.
- [ ] Flip **SPEC-200 + PLAN-200 → Done**; update the specs/plans indexes.
- [ ] Produce the **PT-BR HTML lesson** `docs/lessons/SPEC-200-aula.html` via the
      **frontend-lesson-writer** subagent (product-focused — what the foundation/shell enables and how
      the guards become visible UI, not a React tutorial).
- [ ] Optional design follow-up: add `copilot-fab`/`copilot-panel` to `design-system.md` and drop chat from
      the nav mention (keeps the DS source honest with the floating-widget decision).

#### Deliverables
- Docs updated, spec closed, lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| `api/openapi.yaml` has gaps/imprecise schemas the generator can't type well | Medium | Generate early (Phase 3); log spec gaps as backend follow-ups (out of SPEC-200 scope) — do **not** hand-write DTOs to paper over them |
| Cross-origin auth cookie complexity | Medium | D1 same-origin proxy (Next rewrites) so the Lax HttpOnly cookie works unchanged; CORS+`SameSite=None` only as fallback |
| Aurora glow performance on low-end devices | Low/Med | Reduced-glow / `prefers-reduced-motion` fallback (Phase 6); glow is decorative and sparing by design |
| Fraunces variable font (opsz/wght axes) misrendering or FOUT | Low | Pin the `opsz`/`wght` axes; `font-display: swap`; consider self-host in Phase 1 |
| Scope creep into feature screens | Medium | Hard rule: SPEC-200 ships **stubbed** routes only; feature content is `SPEC-21x` |
| Money represented as JS float / hand-formatted currency | High | Integer centavos/bps end to end; single `money.ts` edge; lint gate (Phase 6); typed from the contract |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-2001…FR-2007 implemented; SPEC-200 acceptance criteria satisfied.
- [ ] BR-2001 (identity from `/auth/me`), BR-2002 (types from OpenAPI), BR-2003 (money integer), BR-2004 (guards structural) respected.
- [ ] Copilot launcher mounts on every authenticated screen; **no** Chat route in nav.

### Technical Validation
- [ ] `web/` isolated from the Go module; path-scoped CI green; Go build unaffected.
- [ ] Type-drift check green; **no `api/openapi.yaml` change**.
- [ ] Same-origin auth posture works; `401` clears state + redirects.
- [ ] No secrets in the client bundle; no float on money; no order affordances present.

### Quality Validation
- [ ] Vitest/RTL + auth integration + Playwright smoke passing.
- [ ] a11y AA contrast + colorblind-safe gain/loss.
- [ ] Reviewed by **frontend-reviewer** + **react-correctness-reviewer**; docs updated.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-200 acceptance criteria satisfied.
- [ ] Unit/component + auth integration + smoke E2E green in path-scoped CI.
- [ ] `web/` builds and **deploys to the Vercel free tier** from the subdirectory (env-driven config).
- [ ] Design system applied via tokens-as-code; guard components enforce FR-013/FR-014 by construction.
- [ ] **CHANGELOG** updated; `web/README.md` + root pointer added; **`api/openapi.yaml` unchanged** (asserted).
- [ ] **SPEC-200 + PLAN-200 flipped to Done**; specs/plans indexes updated.
- [ ] **PT-BR lesson** `docs/lessons/SPEC-200-aula.html` produced (via **frontend-lesson-writer**).
- [ ] Reviewed by the frontend review agents (**frontend-reviewer** + **react-correctness-reviewer**).
- [ ] Pull Request approved.

---

## 11. Deliverables

### Code Deliverables
- `web/` Next.js app: tokens-as-code + component library, typed API client + query hooks, `money.ts`,
  `stream.ts`, auth/session, the app shell + copilot slot, stubbed routes.
- Path-scoped CI config incl. the OpenAPI type-drift check.

### Infrastructure Deliverables
- Vercel free-tier deploy from `web/` (env-driven API base URL, host-swappable).
- Frontend **harness**: review agents (`frontend-reviewer`, `react-correctness-reviewer`), the
  product-focused `frontend-lesson-writer`, the `prettier-edited` + `on-stop-web` hooks, and
  track-aware `/spec-implement` + `/pr-review`.

### Documentation Deliverables
- `web/README.md` + root pointer, CHANGELOG entry, PT-BR lesson, specs/plans index updates.

---

## 12. Post-Implementation Tasks

### Monitoring
- Watch bundle-size budget in CI; revisit glow-performance fallback with real devices.

### Future Improvements
- Full **light theme** per-component (SPEC-200 drafts tokens only); client error tracking; SSR for public/SEO routes.
- **Chart theming** tokens (Recharts) for the three projection scenarios — needed by SPEC-214.

### Technical Debt
- Any `api/openapi.yaml` schema gaps surfaced during type-gen → backend follow-up tickets.
- Design-system doc: formalize `copilot-fab`/`copilot-panel` and the light-theme component treatments.
