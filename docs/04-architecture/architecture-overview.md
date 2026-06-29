# YieldForge — Architecture Overview

> Status: Draft (v0.1.0) · Last Updated: 2026-06-29
> This document describes the **target architecture** the PRD's success criteria
> require: clean layering, MCP-readiness, and multi-agent-readiness *without major
> redesign*. SPECs and PLANs must conform to it. Decisions are recorded as
> [ADRs](adr/).

---

## 1. Architectural Goals

Driven directly by the [PRD](../01-product/PRD.md):

1. **Explainability is structural** — AI outputs carry structured explanations;
   the explanation gate (FR-013) and non-advice guardrail (FR-014) are
   architectural components, not afterthoughts.
2. **Portfolio facts are computed, not generated** — the LLM reasons *over*
   deterministic portfolio/market facts; it does not invent numbers.
3. **MCP-ready & multi-agent-ready** — the Insight Engine is an interface that can
   be backed by a single LLM call today and a CIO-orchestrated agent fleet later,
   with capabilities exposed as MCP tools — no major redesign (success criteria).
4. **Observable by default** — OpenTelemetry traces span API → application →
   external (DB, LLM, market data), including AI cost/latency.

---

## 2. C4 — Level 1: System Context

```
                          ┌──────────────────────────────┐
                          │        External Sources       │
                          │  • FII market data provider   │
                          │  • BCB SGS (SELIC/CDI/IPCA)   │
   ┌──────────┐           │  • IFIX source                │
   │          │  HTTPS    │  • LLM (Claude / OpenAI)      │
   │ Investor │──────────▶│                                │
   │ (browser)│           └───────────────▲───────────────┘
   └──────────┘                           │
        │                                 │ outbound (ingest / inference)
        ▼                                 │
   ┌─────────────────────────────────────┴───────────────┐
   │                   YieldForge System                  │
   │   Next.js Web App  ──▶  Go Backend  ──▶  PostgreSQL  │
   │                         (+ scheduled ingestion jobs) │
   └──────────────────────────────────────────────────────┘
```

---

## 3. C4 — Level 2: Containers

| Container             | Tech            | Responsibility                                                        |
| --------------------- | --------------- | -------------------------------------------------------------------- |
| **Web App**           | Next.js         | UI: portfolio, profile, dashboard, insights, rebalancing, score, projections, **conversational copilot (chat)**. |
| **API / Backend**     | Go (HTTP/REST)  | Domain logic, use cases, auth, AI orchestration (incl. the **chat copilot**), serving the web app. |
| **Ingestion Worker**  | Go (scheduler)  | Periodic FII + macro data ingestion; decoupled from request path.    |
| **Database**          | PostgreSQL      | Holdings, profile, market data snapshots, generated insights/scores. |
| **LLM Provider**      | Free-tier / local (Gemini · Groq · Ollama) | Reasoning over computed facts (behind the `Insighter` port). Claude/OpenAI are paid drop-in upgrades. |

All containers run via **Docker** locally. The **MVP deploys to a free-forever
host** (e.g. Oracle Cloud Always Free / Fly.io / Render) with managed free-tier
Postgres (Neon / Supabase) — the whole stack runs at **zero cost** (PRD G12).
AWS (ECS/Fargate + RDS) is a documented paid-phase option, not the MVP target.
See [ADR-0002](adr/ADR-0002-tech-stack-and-layering.md) and
[ADR-0003](adr/ADR-0003-zero-cost-and-pluggable-llm.md).

---

## 4. Backend Layering (Clean / Hexagonal)

The Go backend follows a ports-and-adapters layering that matches the PLAN
template's phase order (Domain → Persistence → Application → API).

```
        ┌───────────────────────────────────────────────┐
        │  API Layer (HTTP handlers, DTOs, validation)   │   ← drives
        ├───────────────────────────────────────────────┤
        │  Application Layer (use cases, orchestration)  │
        │  e.g. GeneratePortfolioInsights, ComputeScore  │
        ├───────────────────────────────────────────────┤
        │  Domain Layer (entities, value objects, rules) │   ← no deps outward
        │  Portfolio, Holding, InvestorProfile, Insight  │
        ├───────────────────────────────────────────────┤
        │  Ports (interfaces): HoldingRepository,         │
        │  MarketDataProvider, Insighter, Clock           │
        ├───────────────────────────────────────────────┤
        │  Adapters: Postgres repos, HTTP market-data,    │   ← implements ports
        │  LLM Insighter, MCP servers (future)            │
        └───────────────────────────────────────────────┘
```

**Why this matters for the PRD:** the `Insighter` port is the seam that lets a
single-LLM MVP grow into the multi-agent CIO system (Phase 2) and MCP tools
(Phase 3) without touching domain or application code.

---

## 5. The AI Insight Pipeline (Explainable by construction)

```
 Portfolio + Profile + Market Data
        │ (deterministic computation in the Domain/Application layer)
        ▼
 ┌─────────────────┐   facts (allocations, concentration, yields, macro)
 │  Fact Builder   │ ───────────────────────────────────────────────┐
 └─────────────────┘                                                 ▼
                                                          ┌────────────────────┐
                                                          │   Insighter (port) │
                                                          │ MVP: 1 call to a   │
                                                          │ free/local LLM     │
                                                          │ (Gemini·Groq·Ollama)│
                                                          │ Future: CIO fleet  │
                                                          └─────────┬──────────┘
                                                                    ▼
                                                       structured Insight objects
                                                       { title, explanation, ... }
                                                                    │
                                  ┌─────────────────────────────────┼───────────────┐
                                  ▼                                 ▼               ▼
                       FR-013 Explainability Gate      FR-014 Non-Advice Guard   Persist
                       (reject if no explanation)       (reject buy/sell orders)
                                  │
                                  ▼
                            User-facing output (+ disclaimer)
```

- **Grounding:** the LLM receives computed facts, not raw freedom to invent
  numbers → mitigates hallucination (PRD risk).
- **Gates are middleware:** explainability and non-advice checks wrap every
  `Insighter` call, so they hold for the MVP single call *and* the future agent
  fleet.
- **Free by default, paid-ready:** the MVP `Insighter` adapter targets a free-tier
  or local model; Claude/OpenAI are config-swap upgrades (ADR-0003). Insights are
  **cached per portfolio state** and throttled to stay within free rate limits.

### 5a. Projections (deterministic, no LLM)

Passive-income and net-worth projections (PRD FR-016/FR-017) are **pure
computations** in the Domain/Application layer — the same Fact Builder inputs
(yields, rates, current value) plus a configurable monthly contribution, run across
pessimistic / base / optimistic scenarios. No LLM is involved, so they are
deterministic, reproducible, and free; the LLM may *narrate* them later, but the
numbers come from code. Each result carries its scenario assumptions and an
"estimate, not advice" label.

---

### 5b. Conversational Copilot (chat orchestration seam)

The **conversational copilot** (SPEC-108) adds a chat surface *without* a new reasoning
engine. Each user turn is grounded with a **pre-built fact snapshot** (the same Fact
Builder) plus a bounded window of prior turns, then emitted **only** through the
`Insighter` — so the explainability and non-advice gates hold turn by turn, exactly as
for `/insights`.

```
 user turn (free text) ─┐
                        ▼
  intent routing ─▶ Fact Builder ─▶ insight.InsightRequest{ Facts, Task: chat } ─▶ Insighter (gated) ─▶ reply
 (general | "tenho R$X")                                                                                  │
                                                                              persist thread/message (bounded, clearable)
```

This is deliberately the **bridge into Phase 2**: the chat calls the *same* `Insighter`
port, so swapping the single-LLM turn for a CIO-orchestrated agent fleet (below) is an
adapter change, not a redesign — the conversational surface stays put. Grounding stays a
pre-built fact set in the MVP (deterministic, zero-cost); agentic live MCP tool-calling is
the future evolution. See [ADR-0005](adr/ADR-0005-conversational-copilot-orchestration.md).

---

## 6. Multi-Agent & MCP Readiness (Future Phases)

The same `Insighter` port — already exercised by `/insights` **and the conversational
copilot** — is later implemented by a **CIO orchestrator** that fans out to specialized
agents and aggregates an explainable report, surfaced through the same chat surface:

```
 Insighter (port)
   └── CIO Agent ── Macro Agent ──┐
                ── Fixed Income ───┤  each agent reads facts via MCP tools:
                ── FII Agent ──────┤   Portfolio MCP / Market MCP / News MCP
                ── Risk Agent ─────┘
```

Capabilities are exposed as **MCP servers** so agents (and external MCP clients)
consume them uniformly:

| MCP Server     | Tools                                                              |
| -------------- | ----------------------------------------------------------------- |
| Portfolio MCP  | `get_portfolio`, `get_allocations`, `get_dividends`, `get_asset_details` |
| Market MCP     | `get_selic`, `get_ifix`, `get_fii_data`, `get_macro_data`         |
| News MCP       | `get_news`, `get_fii_reports`, `get_market_events`                |

Because these wrap the **same application use cases** the REST API already uses,
adding MCP is additive (a new adapter), satisfying the "no major redesign"
success criterion.

---

## 7. Data Architecture (high level)

Core persisted aggregates (detailed schemas belong in each SPEC):

- **users / auth** — identity, per-user data isolation.
- **investor_profiles** — risk profile, objectives, horizon.
- **holdings** — FII and fixed-income holdings (polymorphic by asset class).
- **market_data_snapshots** — timestamped FII quotes (price, DY, P/VP, sector,
  last dividend) and macro indicators (SELIC, CDI, IPCA, IFIX).
- **insights / health_scores** — generated outputs with their explanations, kept
  for history, reproducibility, and observability.

Market data is stored as **timestamped snapshots** (append-friendly) to support
freshness checks, last-known-good fallback, and future historical analytics.

---

## 8. Cross-Cutting Concerns

- **Security:** auth on all endpoints; per-user isolation; secrets via a secret
  manager; TLS in transit, encryption at rest; input + prompt-injection validation.
- **Observability:** OpenTelemetry traces/metrics/logs with correlation IDs;
  dedicated metrics for ingestion freshness and AI latency/token-cost.
- **Resilience:** idempotent, transactional ingestion (last-known-good preserved);
  graceful AI degradation (core read paths stay up if the LLM is down).
- **Configuration:** 12-factor; environment-driven; Docker for dev/prod parity.

---

## 9. Open Architecture Decisions

Tracked as ADRs in [`adr/`](adr/). Early decisions still needed:

- Concrete **free** FII market-data provider (and its `MarketDataProvider`
  adapter) — the #1 risk (PRD §14).
- Default **free LLM** (Gemini free tier vs. Groq vs. local Ollama) and the
  prompt/Context-Engineering strategy for grounding (ADR-0003).
- HTTP framework / router choice for the Go backend.
- Free-forever host (Oracle Cloud Always Free vs. Fly.io vs. Render) and free-tier
  Postgres (Neon vs. Supabase). AWS deferred to a paid phase (ADR-0003).
