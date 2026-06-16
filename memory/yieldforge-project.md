---
name: yieldforge-project
description: YieldForge — Go + AI Investment Copilot for Brazilian FIIs/Fixed Income, built with SDD; key binding constraints
metadata:
  type: project
---

YieldForge ("Investment Copilot") is the user's learning + real-use project: a Go
backend + Next.js AI platform analyzing a Brazilian FII / Fixed Income portfolio to
produce **explainable** insights, a Health Score, a Rebalancing Assistant, passive-
income projections (pessimistic/base/optimistic), and net-worth projections.
Built with Spec-Driven Development; docs live in [yield-forge/docs/] (PRD →
SPECs → PLANs → architecture/ADRs).

**Binding constraints to respect in every suggestion:**
- **Zero cost.** MVP must run at R$0/month — free tiers, free-forever hosts, or
  local only. Do NOT default to AWS (its free tier is a 12-month trial) or to
  Claude/OpenAI (no perpetual free tier). Default host: Oracle Cloud Always
  Free / Fly.io / Render; DB: Neon / Supabase free tier; LLM: Gemini free tier /
  Groq / local Ollama. Paid (AWS, Claude, OpenAI) is a documented later upgrade.
- **AI is in the MVP** (user chose this on 2026-06-16) via a free/local LLM behind
  a swappable `Insighter` port — paid providers are drop-in adapters, no redesign.
- **Never financial advice.** AI output gives *areas/considerations* only, never
  specific buy/sell orders; every output is explainable + carries a non-advice
  disclaimer. These are enforced as middleware gates (FR-013/FR-014).
- Architecture is hexagonal/ports-and-adapters, MCP-ready and multi-agent-ready
  from day one (future CIO + Macro/FI/FII/Risk agents over MCP).

Biggest open risk: a **free** FII market-data source (price/DY/P-VP/sector) — no
guaranteed free provider; behind a `MarketDataProvider` port with manual override.
