# ADR-0002 — Tech Stack and Backend Layering

| Field    | Value      |
| -------- | ---------- |
| Status   | Accepted   |
| Date     | 2026-06-16 |
| Deciders | Gabigol    |
| Related  | [PRD §11–§12](../../01-product/PRD.md), [Architecture Overview](../architecture-overview.md) |

## Context

The PRD fixes the principal technologies (Go backend, PostgreSQL, Next.js, Claude/
OpenAI, Docker, AWS, OpenTelemetry) and sets two structural success criteria: the
architecture must support a **future multi-agent system** and **MCP integrations**
*without major redesign*. It also mandates that AI output be **explainable** and
**never financial advice**. We need a backend structure that makes these
properties cheap to uphold as the system grows.

## Decision

**Stack** (per PRD constraints):

- **Backend:** Go, exposing a REST/HTTP API.
- **Database:** PostgreSQL (timestamped market-data snapshots; per-user isolation).
- **Frontend:** Next.js (responsive web).
- **AI:** Claude and/or OpenAI behind an internal `Insighter` port.
- **Infra:** Docker for dev/prod parity; AWS as the deployment target.
- **Observability:** OpenTelemetry (traces, metrics, logs) from day one.

**Layering:** the Go backend uses a **Clean / Hexagonal (ports-and-adapters)**
architecture with strict dependency direction inward:

```
API → Application (use cases) → Domain ⟵ Ports ⟵ Adapters
```

Key ports defined up front: `HoldingRepository`, `MarketDataProvider`,
`Insighter`, `Clock`. The `Insighter` port is the seam for AI: backed by a single
LLM call in the MVP, and by a CIO-orchestrated agent fleet later. Explainability
(FR-013) and non-advice (FR-014) guardrails are implemented as middleware wrapping
the `Insighter` port, so they hold regardless of the backing implementation.

**Ingestion** runs as a scheduled worker decoupled from the request path.

The detailed package/directory layout is intentionally deferred to the first
implementation PLAN (SPEC-001), so it can be validated against real code.

> **Clarification (2026-06-17, via SPEC-001):** the hexagonal *principles* above
> (ports & adapters, dependency inversion, framework-free core) are kept, but the
> code is **organised package-by-feature** (`portfolio`, `marketdata`, `insight`)
> rather than package-by-layer. Each feature package owns its domain types, its
> `service` (use cases), and its **port interfaces**, with concrete adapters as
> sibling subpackages (e.g. `internal/portfolio/postgres`). This is idiomatic Go
> and does not change the dependency rule. See SPEC-001 §3a for the canonical tree.

## Consequences

- **Positive:** the `Insighter` and `MarketDataProvider` ports localize the two
  biggest sources of change (AI strategy, data providers); swapping a provider or
  going multi-agent/MCP is an additive adapter, satisfying the "no major redesign"
  criteria.
- **Positive:** guardrails as middleware guarantee explainability/non-advice
  uniformly across MVP and future agents.
- **Positive:** clean layering maps 1:1 to the PLAN template's phase order
  (Domain → Persistence → Application → API), keeping SDD and code aligned.
- **Cost:** more upfront indirection (interfaces, adapters) than a quick CRUD app —
  accepted deliberately for the project's learning and extensibility goals.
- **Open:** concrete choices for HTTP router, FII data provider, AWS runtime, and
  per-task LLM selection are deferred to their own ADRs.
