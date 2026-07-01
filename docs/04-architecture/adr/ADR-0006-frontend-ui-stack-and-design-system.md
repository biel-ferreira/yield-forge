# ADR-0006 — Frontend UI Stack & Design System

| Field    | Value      |
| -------- | ---------- |
| Status   | Accepted   |
| Date     | 2026-06-30 |
| Deciders | Gabigol    |
| Related  | [PRD §11–§12](../../01-product/PRD.md), [ADR-0002](ADR-0002-tech-stack-and-layering.md), [ADR-0003](ADR-0003-zero-cost-and-pluggable-llm.md), [ADR-0004](ADR-0004-frontend-repository-strategy.md) |

## Context

[ADR-0004](ADR-0004-frontend-repository-strategy.md) fixed **where** the frontend lives
(mono-repo, `web/`, Next.js, OpenAPI-typed client, Vercel free tier) but explicitly
**deferred the concrete UI stack, the OpenAPI generation tooling, and the design-system
approach to "the first frontend SPEC and, where significant, their own ADRs."** With the
backend MVP complete through SPEC-108, the first frontend spec (SPEC-200, App Foundation)
is about to be written, and those deferred choices are cross-cutting: they constrain every
`SPEC-2xx` screen that follows, so they belong in an ADR rather than in one screen's spec.

The PRD already fixes **Next.js (responsive web)** ([PRD §12](../../01-product/PRD.md)).
What remains open, and what this ADR decides, is the layer **inside** Next.js: styling and
component library, how the app consumes the frozen `api/openapi.yaml` contract, where the
reusable component library is authored, and how the two **binding product guards**
(FR-013 explainability, FR-014 non-advice) and the **money-never-float** convention are
upheld on the client — the same properties the backend enforces, now on the wire's other side.

Forces specific to this project:

- **Zero cost is binding** ([ADR-0003](ADR-0003-zero-cost-and-pluggable-llm.md)). Every
  frontend tool must be free/OSS or a free-forever tier; no paid component library, no paid
  design SaaS.
- **The OpenAPI contract is the seam** ([ADR-0004](ADR-0004-frontend-repository-strategy.md)).
  `api/openapi.yaml` is hand-maintained and drift-tested against the router on the backend.
  The frontend should **derive** its types from that one contract, not re-declare them — the
  same lockstep discipline the backend already keeps.
- **Money is never a float, end to end** (CLAUDE.md, BR-1022). The backend serializes money
  as `int64` **centavos** and rates as integer **basis points**. The client must parse and
  carry those as integers and format to a display string only at the render edge — the
  `float64` ban extends across the wire into the UI.
- **The guards are product-defining, not cosmetic** ([PRD §6](../../01-product/PRD.md)). Every
  AI insight carries a human-readable **explanation** (FR-013) and output is never an order,
  always a consideration + **non-advice disclaimer** (FR-014). On the client these must be
  **structural**, not per-screen discipline — a reusable component the render path cannot omit.
- **Solo developer, learning-driven** ([PRD §12](../../01-product/PRD.md)). Favor a small,
  well-documented, mainstream toolchain with copy-in (not lock-in) components over a large
  bespoke design system.
- **Claude Design is available.** The `/design-sync` skill + the design-system project at
  claude.ai/design let the reusable component library be authored and previewed as a synced
  design system, then consumed from `web/`.

## Decision

Adopt the following stack **inside** the Next.js app decided by ADR-0004. All choices are
free/OSS, satisfying the zero-cost posture.

1. **Framework & language: Next.js (App Router) + React + TypeScript, `strict` mode.**
   TypeScript is non-negotiable — it is what makes the OpenAPI-derived types load-bearing.
   The App Router is chosen for its first-class streaming support, which SPEC-215 (chat) needs.

2. **Styling & components: Tailwind CSS + shadcn/ui (Radix primitives).**
   shadcn/ui components are **copied into `web/`** (owned source, MIT), not a versioned runtime
   dependency — no lock-in, fully auditable, and they compose with the Claude Design system
   rather than competing with it. Tailwind gives a single token vocabulary (spacing, color,
   type) that the design tokens below feed.

3. **API client: types generated from `api/openapi.yaml`; no hand-written DTOs.**
   `openapi-typescript` generates TypeScript types from the checked-in spec, and
   `openapi-fetch` provides a tiny typed fetch wrapper over them (both MIT, no heavy codegen
   runtime). **Server state** is owned by TanStack Query (caching, retries, invalidation).
   A `web/`-side generation step + a CI check keep the generated types in lockstep with the
   spec — the client mirror of the backend `openapi_test.go` drift guard. This makes the
   OpenAPI document the **single source of truth for the wire contract on both sides**.

4. **Charts: Recharts (MIT).** The product is data-dense (allocation donuts, projection
   series, health-score gauge). Recharts is a mainstream, dependency-light React charting
   library; a future swap sits behind our own chart components, not scattered across screens.

5. **Design system authored via Claude Design, synced into `web/`.**
   A tokens + core-component library is authored as a Claude Design project and kept in sync,
   **one component at a time** (never wholesale replace), via the `/design-sync` skill. The
   library owns the design tokens and the reusable components — crucially the two that encode
   the binding guards as **first-class UI primitives**:
   - **`InsightCard`** — always renders an explanation slot; an insight without its
     explanation is unrepresentable in the component's contract (FR-013, parse-don't-validate
     applied to UI props).
   - **`NonAdviceDisclaimer`** — a non-optional element on any surface that renders AI output
     (FR-014).
   These mirror the backend's `Gated` `Insighter` decorator: the guard is enforced by
   construction, not by reviewer vigilance.

6. **Money & rates are integers in the UI; format only at the render edge.**
   Centavos and basis points cross the wire as integers (parsed via the typed client, never
   `float64`), are carried as integers through component props and state, and are converted to
   a localized `pt-BR` display string (`R$`, `%`) only inside a small formatting helper at the
   render boundary — the client analogue of `internal/platform/money`. No arithmetic on money
   happens on the client in the MVP; figures are computed by the backend and displayed.

7. **Deployment: Vercel free tier** ([ADR-0004](ADR-0004-frontend-repository-strategy.md)),
   building from `web/`, behind config so the host stays swappable — the same
   provider-behind-a-seam posture the backend keeps.

**Frontend SDD numbering:** frontend capabilities get their own **`SPEC-2xx` tier**
(foundational `SPEC-20x`, feature `SPEC-21x`) with matching `PLAN-2xx`, coexisting with the
backend `SPEC-0xx`/`SPEC-1xx` in the one `docs/` tree. This **refines** ADR-0004's original
wording (which reused `SPEC-1xx`, now occupied by backend features 101–108); the intent —
unified SDD, one source of truth — is unchanged. See [SPEC-2xx tier](../../02-specs/README.md).

**Alternatives considered:**

- **A full component library as a runtime dependency (e.g. MUI/Chakra).** Rejected: heavier
  runtime, harder to bend to the design system, and it competes with rather than composes with
  Claude Design. shadcn/ui's copy-in model keeps the source owned and auditable at zero lock-in.
- **Heavy client-side codegen (e.g. Orval generating hooks + a client).** Rejected for the
  MVP: `openapi-typescript` + `openapi-fetch` + TanStack Query is smaller, more transparent,
  and easier to learn; Orval remains an additive option if the surface grows.
- **Hand-written TypeScript DTOs.** Rejected outright: it re-declares the contract and invites
  exactly the drift the backend spends an entire test guarding against.
- **A CSS-in-JS runtime (styled-components/emotion).** Rejected: Tailwind + owned components is
  lighter, has no runtime style cost, and gives one token vocabulary shared with the design
  system.

## Consequences

- **Positive:** one contract, both sides — types flow from `api/openapi.yaml` to the client,
  so a backend endpoint change surfaces as a TypeScript error, not a runtime surprise.
- **Positive:** the binding guards are structural on the client (`InsightCard`,
  `NonAdviceDisclaimer`), mirroring the backend gate — explainability/non-advice hold by
  construction across the whole stack.
- **Positive:** money-never-float extends unbroken into the UI; no float ever represents a
  balance or rate.
- **Positive:** every choice is free/OSS or a free tier — zero-cost posture intact; the
  design system is reusable across all `SPEC-2xx` screens.
- **Positive:** copy-in components (shadcn/ui) + a swappable host + charts-behind-our-own-
  components keep future swaps additive, consistent with the ports posture.
- **Cost / tradeoff:** two design surfaces to keep coherent — Tailwind tokens in `web/` and
  the Claude Design project — reconciled by treating the design system as the source of tokens
  and syncing incrementally.
- **Cost / tradeoff:** a client-side OpenAPI codegen + drift-check step adds a small amount of
  `web/` tooling; accepted as the price of contract lockstep, and it is also a learning goal.
- **Open:** the auth-session posture across origins (cookie/CORS, building on the SPEC-003
  session model), the exact `web/` internal structure, SSR-vs-CSR per route, and SSE streaming
  wiring for chat are deferred to **SPEC-200 (App Foundation)** and, for chat, SPEC-215.
