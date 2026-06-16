# ADR-0003 — Zero-Cost Infrastructure and Pluggable LLM Provider

| Field    | Value      |
| -------- | ---------- |
| Status   | Accepted   |
| Date     | 2026-06-16 |
| Deciders | Gabigol    |
| Related  | [PRD §10 Cost, §12](../../01-product/PRD.md), [ADR-0002](ADR-0002-tech-stack-and-layering.md) |

## Context

A hard product constraint (PRD G12, Cost NFR): the MVP must run at **R$0/month**.
Two technology choices in the original idea conflict with that:

1. **AWS as the target cloud** — AWS's free tier is a **12-month trial**, not
   free-forever, so it would eventually cost money.
2. **AI via Claude / OpenAI** — neither offers a **perpetual free tier**; both bill
   per token (confirmed against the current Claude API pricing reference). Using
   them as the MVP default would incur cost.

At the same time, the project's learning goals require a real AI Insight Engine,
and a future paid upgrade (to Claude/OpenAI, or to AWS) must not force a redesign —
the "no major redesign" success criterion applies here too.

## Decision

**Run everything on free tiers, free-forever services, or local execution, and
make every paid-capable dependency swappable via configuration.**

- **LLM:** all model access goes through the `Insighter` port (PRD FR-018). The MVP
  default is a **free-tier or local** provider:
  - **Google Gemini** free tier (hosted, Gemini Flash), or **Groq** free tier
    (fast Llama inference), or **Ollama** running locally ($0, no rate limit).
  - Claude / OpenAI are **drop-in adapters** for a later paid phase — no domain or
    application changes. Explainability (FR-013) and non-advice (FR-014) guardrails
    wrap the port, so they hold for every provider.
- **Database:** PostgreSQL on a **managed free tier** (e.g. Neon / Supabase) or
  self-hosted in Docker locally.
- **App / compute host:** a **free-forever** option (e.g. Oracle Cloud Always Free,
  Fly.io, Render) instead of AWS. AWS becomes a documented paid-phase option.
- **Observability:** OpenTelemetry (free SDK) exporting to a **free-tier or
  self-hosted** backend (Grafana Cloud free tier / Jaeger / Prometheus).
- **Market data:** BCB SGS API (free) for macro; the FII provider must have a free
  access path, behind the `MarketDataProvider` port, with manual override as
  fallback.

**Cost-safety rules:** free LLM rate limits are respected via caching (insights
keyed to portfolio state) and throttling; exceeding any free tier must degrade
gracefully and **never incur a charge silently**.

## Consequences

- **Positive:** the project meets its zero-budget constraint while still shipping a
  real AI Insight Engine, Rebalancing Assistant, and Health Score.
- **Positive:** every cost-bearing dependency (LLM, DB host, app host, market data)
  is a config/adapter swap, so moving to a paid tier later is additive — consistent
  with ADR-0002's ports-and-adapters seam.
- **Positive:** local Ollama guarantees the AI features keep working even if every
  hosted free tier disappears.
- **Cost / tradeoff:** free-tier LLMs have lower rate limits and may produce
  lower-quality reasoning than Claude/OpenAI; mitigated by caching, grounding
  insights in computed facts (ADR-0002), and the option to upgrade.
- **Operational:** more provider abstractions and config surface than hardcoding a
  single vendor — accepted deliberately for the zero-cost and learning goals.
- **Open:** concrete picks for the free FII data provider, the default free LLM,
  and the free-forever host are tracked as their own decisions.
