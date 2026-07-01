# SPEC-200 — Frontend App Foundation

## 1. Document Information

| Field        | Value                                                                 |
| ------------ | --------------------------------------------------------------------- |
| Feature Name | Frontend App Foundation                                               |
| Feature ID   | SPEC-200                                                              |
| Version      | 1.0.0                                                                 |
| Status       | Approved                                                              |
| Author       | Gabigol                                                              |
| Last Updated | 2026-07-01                                                            |
| Related PRD  | [§11–§12](../01-product/PRD.md); [ADR-0004](../04-architecture/adr/ADR-0004-frontend-repository-strategy.md), [ADR-0006](../04-architecture/adr/ADR-0006-frontend-ui-stack-and-design-system.md) |

---

## 2. Overview

### Purpose

Establish the Next.js web client under `web/` and everything every subsequent frontend
screen (`SPEC-21x`) depends on: the app shell/layout, the **typed API client generated from
`api/openapi.yaml`**, authenticated-session handling against SPEC-003, the **design-system
integration** (Claude Design / `/design-sync`) with the guard-bearing core components, and the
`pt-BR` money/rate formatting edge. This spec ships **no product feature screen** — it is the
frontend's `SPEC-001`: the seam the features plug into.

### Business Value

The backend MVP (SPEC-001…108) exposes value only through an API. This foundation turns that
contract into a usable, consistent, zero-cost web surface, and does so in a way that makes the
binding product guards (FR-013 explainability, FR-014 non-advice) and the money-never-float
convention **structural on the client** — so every feature built on top inherits them for free.

### Success Criteria

- A Next.js + TypeScript app runs under `web/` with its own toolchain, isolated from the Go
  module (ADR-0004 layout).
- The API client's types are **generated from `api/openapi.yaml`**; a `web/` drift check fails
  if the generated types diverge from the committed spec (client mirror of `openapi_test.go`).
- A user can **log in, stay authenticated across reloads, and log out** against the SPEC-003
  session cookie; unauthenticated access to a protected route redirects to login.
- The design system exists as a synced Claude Design project and is consumed from `web/`,
  including `InsightCard` (non-optional explanation slot) and `NonAdviceDisclaimer`.
- Money/rates render as `pt-BR` strings via a single formatting helper; **no `float64`/`number`
  ever represents a balance or rate through arithmetic** — integers in, formatted string out.
- `web/` builds and deploys to the Vercel free tier from the subdirectory.

---

## 3. Functional Requirements

### FR-2001 — Next.js app scaffold under `web/`

The frontend is a self-contained Next.js (App Router) + React + TypeScript (`strict`) app under
`web/`, with its own `package.json`, lockfile, ESLint/format config, and Tailwind + shadcn/ui,
not part of the Go module.

#### Acceptance Criteria

- [ ] `web/` builds (`next build`) and lints clean in isolation.
- [ ] `node_modules/` and build output are git-ignored; the Go build is untouched by `web/**`.
- [ ] Tailwind + shadcn/ui configured; base tokens wired to the design system (FR-2004).

### FR-2002 — Typed API client from the OpenAPI contract

The client's request/response types are **generated** from `api/openapi.yaml`
(`openapi-typescript`), consumed via a typed fetch wrapper (`openapi-fetch`), with server state
owned by TanStack Query. There are **no hand-written DTOs**.

#### Acceptance Criteria

- [ ] A `web/` script regenerates types from `api/openapi.yaml`.
- [ ] A CI/check step fails if committed generated types differ from a fresh generation (drift
      guard, mirroring the backend `openapi_test.go`).
- [ ] Money/rate fields are typed and carried as **integers** (centavos / bps); no float parse.

### FR-2003 — Authenticated session handling (SPEC-003)

The app authenticates against the SPEC-003 endpoints (`/auth/login`, `/auth/logout`,
`/auth/me`) using the session cookie; identity is **never** taken from client state as truth —
`/auth/me` is the source. Protected routes require a session.

#### Acceptance Criteria

- [ ] Login → session cookie set → `/auth/me` resolves the user → app renders authenticated.
- [ ] Reload preserves the session; logout clears it and returns to the login route.
- [ ] Unauthenticated access to a protected route redirects to login (no protected data fetched).
- [ ] Cross-origin cookie/CORS posture documented and working against the backend.

### FR-2004 — Design system + guard-bearing core components

A tokens + core-component library is authored as a Claude Design project and synced into `web/`
via `/design-sync`, one component at a time. It includes the two guard primitives.

#### Acceptance Criteria

- [ ] `InsightCard` cannot render without an explanation slot (FR-013 by construction).
- [ ] `NonAdviceDisclaimer` is present on any surface rendering AI output (FR-014).
- [ ] Core primitives (button, card, input, table, badge, gauge, chart wrappers) available and
      documented as design-system cards.

### FR-2005 — Money & rate formatting edge

A single formatting helper converts integer centavos → `R$` and integer bps → `%` in `pt-BR`.
No money arithmetic happens on the client in the MVP.

#### Acceptance Criteria

- [ ] `formatCentavos(1234567)` → `R$ 12.345,67`; `formatBps(1050)` → `10,50%` (exact, table-tested).
- [ ] Lint/convention forbids `float`-style handling of money fields; values stay integer end to end.

### FR-2006 — App shell, routing & error/loading states

A responsive app shell (sidebar nav + top bar, authenticated layout) with route-level
loading and error boundaries and a consistent `{ "error": "..." }` envelope handling.
The shell also exposes a **global overlay slot** for the floating copilot launcher.

#### Acceptance Criteria

- [ ] Authenticated layout (248px sidebar + top bar + content) with navigation to the
      (stubbed) feature routes — Painel, Carteira, Insights, Saúde, Projeções, Perfil.
      There is **no Chat route**: the copilot is a global floating widget.
- [ ] The shell provides a **global overlay slot** that mounts the floating copilot
      launcher on every authenticated screen (the widget itself is implemented in
      SPEC-215; SPEC-200 only owns the mount point + open/closed shell state).
- [ ] Loading and error states render from a shared pattern; API `{"error":...}` surfaces cleanly.
- [ ] Responsive at mobile + desktop breakpoints (sidebar → bottom tabs on mobile).

### FR-2007 — Zero-cost deploy from `web/`

The app deploys to the Vercel free tier from the `web/` subdirectory, behind config
(API base URL), host swappable.

#### Acceptance Criteria

- [ ] Documented build/deploy config for the free host from the subdirectory.
- [ ] API base URL and any client config are env-driven; no secrets in the client bundle.

---

## 4. User Flows

### Main Flow (authenticated session)

1. User opens the app → unauthenticated → redirected to login.
2. User submits credentials → backend sets the session cookie → app calls `/auth/me`.
3. `/auth/me` resolves the user → authenticated shell renders with navigation.
4. User reloads → session persists via cookie; user logs out → session cleared → login.

### Alternative Flow (expired/invalid session)

1. A protected request returns `401` → app clears local auth state → redirects to login.
2. No protected data is rendered from stale state.

---

## 5. Business Rules

### BR-2001 — Identity from the server, never the client

The authenticated user is whatever `/auth/me` (SPEC-003 cookie) resolves — the client never
trusts a locally-stored user id as authoritative. Mirrors the backend's identity-from-context rule.

### BR-2002 — Contract from OpenAPI, not hand-written

All wire types derive from `api/openapi.yaml`. Divergence is a build failure, not a runtime bug.

### BR-2003 — Money/rates are integers on the client

Centavos and basis points cross the wire and flow through state as integers; formatting to a
display string happens only at the render edge. No `float64` equivalent ever represents money.

### BR-2004 — The guards are structural

`InsightCard` requires an explanation; AI surfaces require `NonAdviceDisclaimer`. The component
contracts make an unexplained insight or an order-like surface unrepresentable.

---

## 6. Domain Model

Not applicable — the frontend holds **no domain model of its own**. It renders types derived
from the backend contract (`api/openapi.yaml`). Client-side state is UI/session state only.

---

## 7. API Contract

This spec **declares no new endpoints**. It consumes the existing, documented surface:
SPEC-003 auth (`POST /auth/login`, `POST /auth/logout`, `GET /auth/me`) and — via the
`SPEC-21x` features — the rest of `api/openapi.yaml`. No `api/openapi.yaml` change belongs to
this spec.

---

## 8. Data Model

Not applicable — no new tables, no persistence. Session persistence is the SPEC-003 cookie;
client caches are ephemeral (TanStack Query).

---

## 9. Edge Cases

### Backend unreachable / 5xx

Show a shared error state; never render a half-populated screen from stale cache.

### Session expires mid-session

A `401` on any protected call clears auth state and redirects to login without leaking data.

### OpenAPI regenerated after a backend change

The drift check fails until types are regenerated, forcing the client back into lockstep.

### AI output surface without a disclaimer

Unreachable by construction — the component contract requires `NonAdviceDisclaimer`; a missing
explanation on `InsightCard` is a type error.

---

## 10. Security Requirements

### Authentication

Session via the SPEC-003 `HttpOnly` cookie; the client never reads the token. `/auth/me` is the
authority for identity.

### Authorization

Protected routes gate on an authenticated session; the backend remains the enforcement point
(the client cannot be the security boundary). Cross-origin requests use the documented CORS +
cookie posture.

### Data Protection

No secrets in the client bundle; only a public API base URL is configured. No PII logged in the
browser console.

---

## 11. Observability

### Metrics

- Client build size budget (bundle) tracked in CI (informational).

### Logs

- Structured client-side error reporting is **out of scope** for the MVP (deferred); rely on the
  backend's OTel (SPEC-004) for server-side traces of the API calls the client makes.

### Traces

- The backend already emits route-named spans (SPEC-004); no client tracing in the MVP.

---

## 12. Testing Strategy

### Unit Tests

- Money/rate formatting helper (table-driven, exact `pt-BR` output).
- OpenAPI drift check (generated types match the committed spec).
- Guard-component contracts (`InsightCard` requires explanation; disclaimer presence).

### Integration Tests

- Auth flow against a running backend: login → `/auth/me` → protected route → logout.
- `401` handling redirects and clears state.

### E2E Tests

- Smoke: unauthenticated redirect, login, authenticated shell renders, logout — one happy-path
  E2E (Playwright, free/OSS), gated so it skips cleanly without a running backend.

---

## 13. Definition of Done

- [ ] Functional requirements (FR-2001…FR-2007) implemented.
- [ ] Acceptance criteria satisfied.
- [ ] Unit tests passing; auth integration test passing against a local backend; smoke E2E green.
- [ ] Design system synced (Claude Design) with the guard components; documented.
- [ ] `web/` builds + deploys to the free host from the subdirectory.
- [ ] Docs updated: this SPEC → Done, PLAN-200 done, CHANGELOG entry, and the PT-BR lesson
      `docs/lessons/SPEC-200-aula.html`. (No `api/openapi.yaml` change — this spec adds no endpoint.)
- [ ] Code reviewed.

---

## 14. Open Questions

1. **SSR vs CSR per route** — which routes (if any) need server rendering vs. a pure client
   fetch? Default CSR for authenticated data; revisit for any public/SEO route.
2. **Cookie/CORS across origins** — exact posture if `web/` (Vercel) and the API are on
   different origins (SameSite, credentials, allowed origins). Resolve during implementation.
3. **SSE streaming plumbing for chat** — SPEC-215 needs token streaming; the client transport
   should be built here so chat is not a rework. Confirm the backend streaming shape first.
4. **Auth UI ownership** — is the login/register screen part of this foundation or a thin
   `SPEC-201`? Currently folded into FR-2003; split out if it grows.
