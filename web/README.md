# YieldForge Web (`web/`)

The Next.js web client for YieldForge — the app foundation (SPEC-200). Backend and frontend
share one repo ([ADR-0004](../docs/04-architecture/adr/ADR-0004-frontend-repository-strategy.md));
the stack is fixed by [ADR-0006](../docs/04-architecture/adr/ADR-0006-frontend-ui-stack-and-design-system.md).
Conventions live in [`CLAUDE.md`](CLAUDE.md) (it inherits the root one).

**Stack:** Next.js 16 (App Router) · React 19 · TypeScript `strict` · Tailwind CSS v4
(CSS-first `@theme`) · the **Aurora** design system · a typed API client generated from
[`../api/openapi.yaml`](../api/openapi.yaml) · TanStack Query · Vitest + Playwright.

## Prerequisites

- **Node ≥ 20** (`.nvmrc` pins 20). Package manager: **npm**.
- For anything that talks to the API (login, data), the **backend must be running** — the Go
  API on `:8080` and Postgres (from the repo root: `task docker-up`, or `task run` + a local DB).

## Setup

```bash
cd web
npm install
cp .env.example .env.local   # set API_PROXY_TARGET if the API isn't on :8080
npm run dev                  # http://localhost:3000
```

The browser calls the app's **own origin** under `/api/*`; Next.js rewrites those to
`API_PROXY_TARGET` (the backend). This keeps the SPEC-003 `HttpOnly` session cookie same-origin —
no CORS (D1 in [PLAN-200](../docs/03-plans/PLAN-200-app-foundation.md)).

## Commands

| Need | Command |
| ---- | ------- |
| Dev server | `npm run dev` |
| Production build | `npm run build` |
| Type-check | `npm run typecheck` |
| Lint | `npm run lint` |
| Format / check | `npm run format` / `npm run format:check` |
| **Regenerate the API types** from `../api/openapi.yaml` | `npm run gen:api` |
| **API drift check** (types vs the contract) | `npm run check:api` |
| Unit/component tests | `npm run test` (`vitest run`) |
| E2E smoke (needs `npx playwright install chromium` + the backend up) | `npm run test:e2e` |

**Quality gate before done:** `npm run typecheck && npm run lint && npm run check:api && npm run test && npm run build`.
The path-scoped [`web-ci`](../.github/workflows/web-ci.yml) workflow runs the same on every `web/**` change.

## The API contract is generated, never hand-written

Request/response types come from `../api/openapi.yaml` via `openapi-typescript` → `lib/api/schema.ts`
(committed). If the contract changes, run `npm run gen:api` and commit; `check:api` fails the build
if the committed types drift — the client mirror of the backend `openapi_test.go`.

## Structure

```
web/
├── app/                 # App Router: (app) protected shell + routes, login/register, styleguide
├── components/          # ui/ primitives, the guard components, shell/ (sidebar, top bar, copilot)
├── lib/                 # api/ (typed client + generated schema), money.ts, stream.ts, auth/, shell/
├── e2e/                 # Playwright smoke
└── scripts/             # gen/check API types
```

## Deploy

Vercel free tier, building from this `web/` subdirectory (env-driven `API_PROXY_TARGET`,
host-swappable). See [ADR-0004](../docs/04-architecture/adr/ADR-0004-frontend-repository-strategy.md).
