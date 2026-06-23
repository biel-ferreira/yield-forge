# ADR-0004 — Frontend Repository Strategy (Mono-repo)

| Field    | Value      |
| -------- | ---------- |
| Status   | Proposed   |
| Date     | 2026-06-23 |
| Deciders | Gabigol    |
| Related  | [PRD §12](../../01-product/PRD.md), [ADR-0002](ADR-0002-tech-stack-and-layering.md), [ADR-0003](ADR-0003-zero-cost-and-pluggable-llm.md) |

## Context

The PRD fixes **Next.js (responsive web)** as the frontend ([PRD §12](../../01-product/PRD.md)),
but the MVP roadmap to date is entirely backend: the foundational specs
(`SPEC-001…006`) and the feature specs (`SPEC-1xx`) deliver a REST/HTTP API and the
hexagonal Go domain behind it. No spec yet describes a user interface, and the
repository currently holds only the Go module (`go.mod` at the root, `cmd/`,
`internal/`, `docs/`). Before the first UI spec is written we must decide **where
the frontend code lives** relative to the backend, because that choice shapes the
SDD flow, CI, and deployment for everything that follows.

Forces specific to this project:

- **Solo developer, learning-driven** ([PRD §12](../../01-product/PRD.md)): coordination
  overhead between repositories is a real, recurring cost for one person.
- **SDD with a single source of truth**: the whole `docs/` tree (PRD, SPECs, PLANs,
  ADRs, PT-BR lessons) lives in this repository, and the PRD → SPEC → PLAN → CODE
  flow is the backbone of the project. Frontend work will need its own SPECs/PLANs
  that must coexist with the backend ones.
- **The API is the architectural seam, not the frontend**: per [ADR-0002](ADR-0002-tech-stack-and-layering.md)
  and the MCP / multi-agent posture, the backend is stateless and the REST API is
  the contract. The Next.js app is *one client among future others* (a possible
  native app, MCP clients). This is a **runtime** boundary and does not, by itself,
  dictate a repository boundary.
- **Mixed toolchains**: Go (modules, `golangci-lint`, `go test`) vs. Next.js
  (npm/pnpm, `node_modules`, ESLint). This is the genuine friction of co-locating
  the two.
- **Zero-cost deployment** ([ADR-0003](ADR-0003-zero-cost-and-pluggable-llm.md)): the
  backend targets a free-forever host; the frontend targets a free Next.js host
  (e.g. Vercel free tier) — both can build and deploy from a subdirectory, so a
  single repository does not block independent deploys.

## Decision

**Adopt a mono-repo.** The Next.js frontend lives in this repository under a
top-level `web/` directory, alongside the existing Go code; backend and frontend
stay in one repo, one issue tracker, and one SDD `docs/` tree.

- **Layout:** the Go module stays at the repository root (no churn to existing
  imports, `cmd/`, `internal/`). The frontend is added as a self-contained `web/`
  directory with its **own** `package.json`, lockfile, and lint/test config. `web/`
  is **not** part of the Go module; `node_modules/` and build output are
  git-ignored.

  ```
  yield-forge/
  ├── cmd/            # Go API + migrate
  ├── internal/       # hexagonal domain + adapters
  ├── migrations/
  ├── web/            # Next.js app (own package.json / toolchain)
  ├── docs/           # SDD: backend AND frontend SPECs/PLANs/ADRs/lessons
  ├── go.mod
  └── Taskfile.yml
  ```

- **CI scoped by path:** pipelines trigger per changed area — a change touching only
  `web/**` runs the frontend build/lint/test and does not rebuild the Go backend,
  and vice-versa. This recovers most of the independence of separate repos without
  the overhead of two.

- **API contract is explicit:** the boundary between `web/` and the backend is the
  REST API, described by a checked-in **OpenAPI** document that the backend serves
  and the frontend consumes (typed client generated from it). The contract — not
  shared source — is what keeps the two halves honest; this preserves the
  "frontend is just a client" property and keeps the door open for additional
  clients (native app, MCP).

- **SDD stays unified:** frontend capabilities get their own `SPEC-1xx` / matching
  `PLAN-1xx` in `docs/`, and close with a PT-BR lesson, exactly like backend specs.
  One `docs/` tree remains the single source of truth.

- **Deployment stays independent and zero-cost:** the frontend deploys from `web/`
  to its free host; the backend deploys from the root to its free-forever host
  ([ADR-0003](ADR-0003-zero-cost-and-pluggable-llm.md)). Same repo, separate
  pipelines.

**Alternative considered — two repositories** (`yield-forge` + `yield-forge-web`):
rejected for the MVP. It yields a pristine Go repo and naturally independent
deploys, but for a solo developer it doubles the operational surface (two trackers,
two CI configs), splits the SDD documents (risking doc drift / "split-brain"), and
makes a cross-stack change — a new endpoint plus the UI that consumes it — span two
repositories and two PRs, inviting API-contract drift. None of those costs buys a
property the mono-repo cannot achieve with path-scoped CI and an OpenAPI contract.

A second sub-alternative — restructuring the backend into `backend/` alongside
`web/` — was also rejected: it churns every existing path and import for no benefit
the root-level `web/` directory does not already provide.

## Consequences

- **Positive:** lowest coordination overhead for a solo developer — one clone, one
  issue tracker, one CI configuration.
- **Positive:** SDD keeps a single source of truth; frontend and backend SPECs,
  PLANs, ADRs, and lessons live together and evolve in lockstep.
- **Positive:** cross-stack changes (endpoint + consuming UI) land in one atomic PR,
  which keeps the API contract honest by construction.
- **Positive:** the "frontend is one client of the API" property is preserved via
  the OpenAPI contract, so additional clients (native app, MCP) remain additive.
- **Cost / tradeoff:** two toolchains coexist in one repo (Go + Node), requiring a
  slightly more complex `.gitignore`, path-scoped CI, and separate lint/test
  invocations. Accepted deliberately; it is also a learning goal.
- **Cost / tradeoff:** the repository is no longer pure-Go, which slightly muddies
  language-level tooling that assumes a single ecosystem at the root. Contained by
  isolating everything frontend under `web/`.
- **Operational:** deployment requires per-subdirectory build configuration on the
  respective free hosts; documented when the first UI spec is implemented.
- **Open:** the concrete `web/` internal structure, the OpenAPI generation/tooling,
  the frontend's free host, and the auth-session integration across origins
  (cookie/CORS posture, building on [ADR-0002](ADR-0002-tech-stack-and-layering.md)
  and the SPEC-003 session model) are deferred to the first frontend SPEC and, where
  significant, their own ADRs.
