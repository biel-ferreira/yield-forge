# CLAUDE.md — YieldForge Web (`web/`)

Frontend conventions for the Next.js client. This file is **nested memory**: it loads only
when working under `web/`. It **inherits** the root [`../CLAUDE.md`](../CLAUDE.md) — the
**binding product constraints** (explainability, non-advice, facts-are-computed, zero-cost)
and the SDD working agreement apply here unchanged. This file records how the client
**upholds** them and the conventions specific to React/TypeScript.

Source of truth for the stack: [ADR-0006](../docs/04-architecture/adr/ADR-0006-frontend-ui-stack-and-design-system.md)
(UI stack & design system) over [ADR-0004](../docs/04-architecture/adr/ADR-0004-frontend-repository-strategy.md)
(mono-repo `web/`, Vercel). First frontend spec: SPEC-200 (App Foundation).

## Stack (ADR-0006 — do not re-litigate per screen)

- **Next.js (App Router) + React + TypeScript `strict`.** TS strict is non-negotiable — it is
  what makes the OpenAPI-derived types load-bearing. App Router is chosen for streaming (SPEC-215 chat).
- **Styling: Tailwind CSS** — one token vocabulary (spacing/color/type) fed by the Aurora design tokens.
- **Components: shadcn/ui (Radix)** — **copied into `web/` (owned MIT source), never a runtime
  dependency.** No lock-in; auditable. They compose with Aurora, they don't replace it.
- **API client: `openapi-typescript` + `openapi-fetch`** over the checked-in
  [`../api/openapi.yaml`](../api/openapi.yaml). **No hand-written DTOs** (see below).
- **Server state: TanStack Query** (caching, retries, invalidation). Do not hand-roll fetch/cache.
- **Charts: Recharts (MIT)**, always wrapped behind our own chart components so a future swap
  is additive — the ports posture, on the client.
- **Design system: Aurora**, authored in Claude Design, synced **one component at a time** via
  `/design-sync` (never wholesale replace). Design mocks live in
  [`../docs/05-design/ds/`](../docs/05-design/ds/).

## Binding guards are structural (never per-screen discipline)

These mirror the backend's `Gated` `Insighter` decorator — enforced **by construction**, not by
reviewer vigilance. Parse-don't-validate, applied to component props.

- **`InsightCard` always renders an explanation slot** (FR-013). An insight without its
  human-readable explanation must be **unrepresentable** in the component's prop contract — make
  the explanation a required prop, not an optional one.
- **`NonAdviceDisclaimer` is non-optional on any surface that renders AI output** (FR-014). The
  render path cannot omit it. Never surface a buy/sell order, ticker-to-buy, quantity, or price
  target — only areas/considerations + the disclaimer.

## Money & rates are integers, end to end (BR-1022)

The `float64` ban extends across the wire into the UI.

- Money crosses the wire and flows through props/state as **`int64`-origin centavos**; rates as
  integer **basis points** (1 bp = 0.01%). Parse inbound amounts as integers (typed client /
  `json.Number`-equivalent), **never `float64`**.
- Convert to a localized **`pt-BR`** display string (`R$`, `%`) **only at the render edge**, in a
  small formatting helper — the client analogue of `internal/platform/money`.
- **No money arithmetic on the client** in the MVP. Figures are computed by the backend and
  displayed; the client formats, it does not compute.

## API contract — one source of truth, both sides

- Client types are **generated from `../api/openapi.yaml`**; never re-declare DTOs (that invites
  exactly the drift the backend `openapi_test.go` guards against).
- A `web/`-side generation step + a **drift check** keeps generated types in lockstep with the
  spec — the client mirror of `openapi_test.go`. Regenerate when the contract changes.
- `SPEC-21x` screens **declare no new endpoints** — they consume their backend twin over the
  frozen contract, so they carry no `openapi.yaml` change.
- **Identity comes from the authenticated session, never a request payload** — the client's
  mirror of the backend rule. No client-supplied `user_id`.

## Code conventions

- **Language: English** for code/comments/commits (root rule). The only PT-BR is the design copy
  and the `docs/lessons/*-aula.html` teaching material.
- **Doc comments cite the governing SPEC/BR/FR** they implement — SDD traceability from doc to code.
- **DTOs/wire types are separate from view/domain props** — validate/parse at the fetch edge, as
  the backend keeps HTTP DTOs separate from domain types.
- **Commits: Conventional Commits** (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`), scoped to `web/`.
- **Dependencies: justify any new one** (zero-cost posture, ADR-0003). Prefer the owned/copy-in
  model over runtime lock-in.
- **CHANGELOG:** update the root [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` in the same
  change as notable frontend work.

## Deferred to SPEC-200 (do not invent ahead of the spec)

The following are **open** and are decided in SPEC-200 (App Foundation) — do not hard-code them
here or improvise a structure:

- The exact `web/` internal folder layout and component/file naming.
- Auth-session posture across origins (cookie/CORS over the SPEC-003 session model).
- SSR-vs-CSR per route; SSE streaming wiring for chat (finalized in SPEC-215).
- The concrete toolchain commands (`dev`/`build`/`lint`/`typecheck`/`test`) and the Task/npm
  scripts — added to this file's **Commands** section once SPEC-200 scaffolds them.

## Commands

_To be filled in by SPEC-200 once the toolchain exists (lint/format/typecheck/test/dev/build)._
Until then, the SDD loop is unchanged: `/spec-new`, `/plan-new`, `/spec-implement`, `/pr-review`.
