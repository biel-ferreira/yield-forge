# Product Requirements Document (PRD)

## 1. Document Information

| Field        | Value                                                        |
| ------------ | ------------------------------------------------------------ |
| Product Name | YieldForge — Investment Copilot                              |
| Product Code | YF-COPILOT                                                   |
| Version      | 0.1.0                                                        |
| Status       | Draft                                                        |
| Author       | Gabigol (programacao.blume@gmail.com)                        |
| Last Updated | 2026-06-16                                                   |
| Stakeholders | Product Owner / Engineer (solo), end-investor (primary user) |

---

## 2. Executive Summary

### Product Vision

YieldForge is an **AI-powered personal investment platform** that helps retail
investors **understand, monitor and optimize** their portfolio through
data-driven insights and AI-assisted analysis.

The platform **does not provide financial advice** or direct buy/sell
recommendations. Instead, it analyzes the user's portfolio, market conditions and
investment goals to **generate insights, highlight risks, and suggest areas that
may deserve attention** — always with a clear, human-readable explanation.

It serves a dual purpose:

1. A **real-world tool** the author uses to manage investments in Brazilian FIIs
   and Fixed Income.
2. A **showcase of modern software engineering**: Go backend, AI Engineering,
   multi-agent systems, MCP integrations, Context Engineering, Spec-Driven
   Development, cloud architecture, and observability.

### Problem Statement

Brazilian retail investors juggle multiple disconnected tools to track
investments, monitor allocation, analyze FIIs and fixed income, and follow
macroeconomic indicators (SELIC, IPCA, CDI, IFIX).

Existing platforms mostly **display** information — they do **not reason about the
investor's portfolio as a whole**. There is no single place that understands the
**current portfolio + investor profile + goals + market conditions** together and
produces contextual, explainable insights.

### Proposed Solution

An **Investment Copilot** that:

- Centralizes the user's portfolio (FIIs and Fixed Income to start).
- Periodically ingests market and macroeconomic data.
- Reasons over the *entire* portfolio against the investor's profile and goals.
- Produces **explainable** insights, a **Portfolio Health Score**, and a
  **Rebalancing Assistant** that suggests *areas* (never specific buy orders).
- Is architected from day one to expand into a **multi-agent system** (Macro, Fixed
  Income, FII, Risk, CIO agents) exposed over **MCP** tools.

### Target Audience

Brazilian retail investors who self-manage a portfolio weighted toward **FIIs and
Fixed Income**, are comfortable with digital tools, and want to understand the
"why" behind portfolio decisions rather than receive black-box tips.

---

## 3. Business Context

### Background

The Brazilian FII market is large, dividend-focused, and popular with retail
investors seeking passive income. Fixed income (Tesouro Direto, CDB, and
"caixinhas" like Nubank/Mercado Pago) is the default allocation for most
Brazilians given a historically high interest-rate (SELIC) environment. Investors
constantly weigh fixed income vs. FII yield in light of macro conditions, but no
mainstream tool reasons across both with the investor's personal goals in mind.

### Opportunity

- **Personal:** A genuinely useful tool for the author's own investing.
- **Technical:** A non-trivial, real-world domain to practice Go, AI Engineering,
  multi-agent orchestration, MCP, and cloud architecture end-to-end.
- **Differentiation:** Portfolio-level *reasoning* with *explainability*, not just
  data display.

### Strategic Alignment

Directly advances the author's learning goals (Go, AI Engineering, multi-agent
systems, MCP, Context Engineering, SDD, cloud, observability) while producing a
tool with sustained personal utility — ensuring long-term motivation to maintain
and extend it.

---

## 4. Goals and Objectives

### Primary Goals

- **G1** — Let a user register and continuously monitor a portfolio of FIIs and
  Fixed Income.
- **G2** — Present an accurate dashboard: total invested, estimated current value,
  monthly passive income, growth, and allocation breakdowns.
- **G3** — Automatically keep market and macroeconomic data fresh.
- **G4** — Generate **explainable** AI insights about portfolio concentration,
  allocation, and market context.
- **G5** — Provide a **Rebalancing Assistant** that gives contextual guidance for
  new money — *areas*, never specific buy orders.
- **G6** — Produce an explainable **Portfolio Health Score (0–100)**.
- **G7** — Project **expected passive income** from the current portfolio across
  pessimistic / base / optimistic scenarios.
- **G8** — Project **net-worth growth** over time from current value, ongoing
  income, and a configurable monthly contribution.

### Secondary Goals

- **G9** — Architect for a future **multi-agent system** without major redesign.
- **G10** — Architect for **MCP** integrations that can be added incrementally.
- **G11** — Demonstrate production-grade engineering: clean layering, tests,
  observability (OpenTelemetry), and cloud deployment.
- **G12** — Run the entire stack at **zero cost** (free tiers / free-forever
  services / local execution).

### Success Metrics

| Metric                                   | Target                                              |
| ---------------------------------------- | --------------------------------------------------- |
| Portfolio registration completeness      | User can register 100% of MVP asset types (FII, FI) |
| Dashboard allocation accuracy            | Allocation % matches manual calculation (±0.5%)     |
| Market data freshness                    | FII & macro data no older than 24h                  |
| Insight explainability                   | 100% of insights include a reasoning explanation    |
| Insight relevance (self-rated)           | ≥ 80% of generated insights rated useful by user    |
| Health Score reproducibility             | Same inputs → same score + identical explanation    |
| Rebalancing guidance compliance          | 0 outputs containing a specific buy/sell order      |
| Income projection scenarios              | 3 scenarios (pessimistic/base/optimistic) per run   |
| Net-worth projection                     | Recomputes on contribution change; assumptions shown |
| Backend API p95 latency (non-AI reads)   | < 300 ms                                             |
| Test coverage (domain + application)     | ≥ 80%                                               |
| Infrastructure cost                      | R$0 / month (free tiers, free-forever, or local)    |

---

## 5. Scope

### In Scope (MVP — see §14 for the phased breakdown)

- **Portfolio Management** — register/edit/delete FIIs and Fixed Income holdings.
- **Investor Profile** — risk profile, objectives, investment horizon.
- **Dashboard** — portfolio summary, allocation by asset class, FII sector exposure.
- **Market Data Module** — scheduled ingestion of FII data and macro indicators.
- **AI Insight Engine** — portfolio, allocation, and market-context insights
  (explainable), powered by a **free-tier LLM** (see §12 / ADR-0003).
- **AI Rebalancing Assistant** — contextual guidance for new contributions.
- **Portfolio Health Score** — 0–100 with detailed explanation.
- **Passive Income Projection** — estimated monthly/annual income across
  pessimistic / base / optimistic scenarios, from FII dividend yields and
  fixed-income rates.
- **Net-Worth Projection** — wealth growth over time from current value +
  reinvested income + a configurable monthly contribution.
- Single-user-account experience with authentication.

### Out of Scope (MVP)

- Direct buy/sell execution or brokerage integration (order routing).
- Specific buy/sell **advice** or price targets (forbidden by design — see §6).
- Stocks and ETFs as *fully managed* asset classes (allocation buckets exist in
  the model, but ingestion/analysis is deferred). 
- Automated import from brokerage statements / B3 (manual entry first).
- Tax reporting (IR), DARF calculation, and accounting.
- Mobile native apps (responsive web only).
- Multi-currency accounting beyond BRL (international exposure is referenced as an
  *insight category*, not a managed asset class, in the MVP).
- Real-time/intraday price streaming.

---

## 6. Core Principles (Product Constraints)

These are **binding constraints** that flow down into every SPEC and PLAN.

1. **Explainability First** — every AI-generated insight, score, or suggestion
   **must** include a clear, human-readable explanation. No black-box outputs.
2. **Portfolio-Centric** — the system reasons about the *entire* portfolio, not
   assets in isolation.
3. **Goal-Oriented** — analysis always considers investor profile, risk tolerance,
   time horizon, and financial objectives.
4. **AI as Copilot, not Advisor** — the platform **assists** decisions. It
   **never** provides financial advice or specific buy/sell recommendations. All
   user-facing AI output is framed as *areas/considerations*, with an explicit
   non-advice disclaimer.

---

## 7. User Personas

### Persona 1 — "Rafael", the Self-Directed Dividend Investor

#### Description

32-year-old IT professional in Brazil. Invests monthly (R$1,000–R$3,000), heavily
into FIIs for passive income plus fixed income for safety. Comfortable with apps
and data, but not a finance professional. Currently spreads data across a
spreadsheet, the broker app, and FII news sites.

#### Goals

- See his whole portfolio and real allocation in one place.
- Understand whether he's over-concentrated or under-diversified.
- Decide *where* to direct each month's new contribution.

#### Pain Points

- No tool reasons about his portfolio **as a whole**.
- Hard to tell if his sector mix (logistics vs. paper vs. shopping) is balanced.
- Doesn't know if his fixed-income/FII split fits the current SELIC environment.

---

### Persona 2 — "Carla", the Goal-Driven Long-Term Planner

#### Description

40-year-old planning for retirement and stable passive income over a 15–20 year
horizon. Moderate risk profile. Wants alignment between her portfolio and her
explicit long-term objectives, and a simple signal of overall health.

#### Goals

- Confirm her portfolio is aligned with a retirement/passive-income objective.
- Get a single, trustworthy health indicator she can track over time.
- Receive plain-language explanations she can reason about and trust.

#### Pain Points

- Existing tools show numbers but not *alignment with her goals*.
- No single, explainable "is my portfolio healthy?" answer.
- Distrusts black-box "hot tips".

---

## 8. User Stories

> Stories are grouped into epics that map directly to MVP features. Each epic
> becomes one or more SPECs in [`../02-specs/`](../02-specs/).

### Epic 1 — Portfolio Management

#### User Story

As an investor, I want to **register and maintain my FIIs and fixed-income
holdings** so that **I have a single, accurate view of everything I own**.

#### Acceptance Criteria

- [ ] User can create, edit, and delete an **FII** holding (`ticker`, `quantity`,
      `average_price`).
- [ ] User can create, edit, and delete a **Fixed Income** holding (`name`,
      `institution`, `invested_amount`, `annual_rate`, `maturity_date`,
      `liquidity_type`).
- [ ] Invalid inputs are rejected with clear validation messages (e.g. negative
      quantity, malformed ticker, maturity date in the past for new FI).
- [ ] Holdings persist and are scoped to the authenticated user.

---

### Epic 2 — Investor Profile

#### User Story

As an investor, I want to **define my risk profile, objectives, and horizon** so
that **insights are tailored to my personal situation**.

#### Acceptance Criteria

- [ ] User can set risk profile: `Conservative | Moderate | Aggressive`.
- [ ] User can set one or more objectives: `Retirement | Passive Income | Wealth
      Preservation | Long-Term Growth`.
- [ ] User can set an investment horizon (e.g. 5 / 10 / 20 years).
- [ ] Profile is persisted and consumed by the Insight Engine, Rebalancing
      Assistant, and Health Score.

---

### Epic 3 — Dashboard

#### User Story

As an investor, I want a **dashboard summarizing my portfolio** so that **I
understand my totals, allocation, and sector exposure at a glance**.

#### Acceptance Criteria

- [ ] Summary shows: total invested value, current estimated value, monthly
      passive income, portfolio growth.
- [ ] Allocation is shown by asset class: FIIs, Fixed Income, Stocks, ETFs.
- [ ] FII **sector exposure** is shown: Logistics, Offices, Shopping, Hybrid, Paper.
- [ ] Figures reconcile with the underlying holdings and latest market data.

---

### Epic 4 — Market Data Module

#### User Story

As the system, I want to **periodically collect FII and macroeconomic data** so
that **the dashboard and insights reflect current market conditions**.

#### Acceptance Criteria

- [ ] Per FII: current price, dividend yield, P/VP, sector, last dividend.
- [ ] Macro indicators: SELIC, Inflation (IPCA), CDI, IFIX.
- [ ] Data is refreshed on a schedule and timestamped (freshness ≤ 24h target).
- [ ] Ingestion failures are logged and do not corrupt last-known-good data.

---

### Epic 5 — AI Insight Engine

#### User Story

As an investor, I want **explainable insights about my portfolio** so that **I can
spot concentration, imbalance, and risk without manual analysis**.

#### Acceptance Criteria

- [ ] Generates **Portfolio Insights** (concentration, sector imbalance, excessive
      risk exposure, low diversification).
- [ ] Generates **Allocation Insights** (fixed income below target, excessive
      single-sector exposure, high single-asset concentration).
- [ ] Generates **Market Context Insights** (high-interest environment,
      inflationary environment, favorable scenario for fixed income).
- [ ] **Every** insight includes a clear explanation of *why* it was raised.
- [ ] No insight contains a specific buy/sell recommendation.

---

### Epic 6 — AI Rebalancing Assistant

#### User Story

As an investor, I want to say **"I have R$X to invest this month"** and receive
**contextual guidance on where to focus** so that **I can allocate new money
thoughtfully — without being told exactly what to buy**.

#### Acceptance Criteria

- [ ] Accepts a contribution amount as input.
- [ ] Analyzes current portfolio, diversification, goals, and market conditions.
- [ ] Outputs **suggested areas** (e.g. Fixed Income, Logistics FIIs,
      International Exposure) plus **reasoning**.
- [ ] **Never** outputs a specific buy order (no ticker + quantity to purchase).
- [ ] Output explicitly carries a non-advice disclaimer.

---

### Epic 7 — Portfolio Health Score

#### User Story

As an investor, I want a **0–100 health score with a detailed explanation** so
that **I have a single, trustworthy, trackable signal of portfolio health**.

#### Acceptance Criteria

- [ ] Score is computed from: diversification, concentration, liquidity, goal
      alignment, risk exposure.
- [ ] Output includes the numeric score **and** a detailed breakdown explaining
      each contributing factor.
- [ ] Score is deterministic for identical inputs (reproducible).

---

### Epic 8 — Passive Income Projection

#### User Story

As an investor, I want to **see an estimate of the passive income my portfolio
will generate** — from pessimistic to optimistic — so that **I understand the
income my FIIs and fixed income are likely to produce and can plan around it**.

#### Acceptance Criteria

- [ ] Estimates monthly and annual passive income from FII dividend yields and
      fixed-income annual rates over the current holdings.
- [ ] Presents **three scenarios** — pessimistic / base / optimistic — with the
      assumptions behind each (e.g. yield haircut/uplift, rate changes) shown.
- [ ] Figures reconcile with the holdings and latest market data; the calculation
      is deterministic and reproducible.
- [ ] Clearly labelled as an estimate, not a guarantee (non-advice).

---

### Epic 9 — Net-Worth Projection

#### User Story

As an investor, I want to **project how my net worth grows over time** based on
current value, ongoing income, and a **configurable monthly contribution**, so
that **I can see the long-term trajectory toward my goals**.

#### Acceptance Criteria

- [ ] Projects portfolio/net-worth value over a configurable horizon (e.g. 5 / 10 /
      20 years), starting from current estimated value.
- [ ] User can configure the **monthly contribution**; the projection recomputes.
- [ ] Incorporates reinvested income and the pessimistic/base/optimistic scenarios
      from Epic 8; assumptions are shown.
- [ ] Output is suitable for charting over time and labelled as an estimate.

---

## 9. Functional Requirements

> IDs are referenced by SPECs. NF requirements are in §10.

- **FR-001 — Holding Management (FII):** CRUD for FII holdings with fields
  `ticker`, `quantity`, `average_price`, scoped to the authenticated user.
- **FR-002 — Holding Management (Fixed Income):** CRUD for fixed-income holdings
  with fields `name`, `institution`, `invested_amount`, `annual_rate`,
  `maturity_date`, `liquidity_type` (e.g. CDB, Tesouro Direto, daily-liquidity
  "caixinha").
- **FR-003 — Investor Profile:** Persist risk profile, objectives, and investment
  horizon; expose them to all analysis components.
- **FR-004 — Portfolio Summary:** Compute total invested, current estimated value,
  monthly passive income, and growth.
- **FR-005 — Allocation Breakdown:** Compute allocation by asset class (FII, Fixed
  Income, Stocks, ETFs) and FII sector exposure (Logistics, Offices, Shopping,
  Hybrid, Paper).
- **FR-006 — Market Data Ingestion (FII):** Periodically fetch and store current
  price, dividend yield, P/VP, sector, and last dividend per FII.
- **FR-007 — Macro Data Ingestion:** Periodically fetch and store SELIC, Inflation
  (IPCA), CDI, and IFIX.
- **FR-008 — Portfolio Insights:** Generate explainable insights on concentration,
  sector imbalance, risk exposure, and diversification.
- **FR-009 — Allocation Insights:** Generate explainable insights on
  target-vs-actual allocation and single-sector/single-asset concentration.
- **FR-010 — Market Context Insights:** Generate explainable insights tying macro
  conditions to the portfolio (e.g. high-rate environment).
- **FR-011 — Rebalancing Assistant:** Given a contribution amount, output
  suggested allocation *areas* with reasoning and no specific buy order.
- **FR-012 — Portfolio Health Score:** Compute a 0–100 score from diversification,
  concentration, liquidity, goal alignment, and risk exposure, with a detailed
  explanation.
- **FR-013 — Explainability Guarantee:** Every AI output (insight, suggestion,
  score) carries a structured explanation. Outputs without an explanation are
  rejected before reaching the user.
- **FR-014 — Non-Advice Guardrail:** All AI outputs are validated to exclude
  specific buy/sell instructions and carry a non-advice disclaimer.
- **FR-015 — Authentication:** Users authenticate; all portfolio data is isolated
  per user.
- **FR-016 — Passive Income Projection:** Estimate monthly and annual passive
  income from FII dividend yields and fixed-income rates, across pessimistic /
  base / optimistic scenarios, with the assumptions for each scenario exposed.
  Deterministic and reproducible.
- **FR-017 — Net-Worth Projection:** Project portfolio value over a configurable
  horizon from current value + reinvested income + a configurable monthly
  contribution, using the same scenario set as FR-016, with assumptions shown and
  output suitable for time-series charting.
- **FR-018 — Pluggable LLM Provider:** All LLM access goes through a single
  internal port (`Insighter`) so the provider is swappable via configuration. The
  MVP uses a **free-tier or local LLM**; paid providers are drop-in upgrades with
  no domain/application changes (see ADR-0003).

---

## 10. Non-Functional Requirements

### Performance

- Non-AI read endpoints (dashboard, holdings) respond in **< 300 ms p95**.
- AI insight generation responds in **< 10 s p95** (async/streamed where helpful).

### Scalability

- Stateless backend services, horizontally scalable behind a load balancer.
- Market-data ingestion runs as scheduled jobs decoupled from request handling.
- Architecture must accommodate the future multi-agent system and MCP servers
  **without major redesign** (a first-class success criterion).

### Security

- All endpoints require authentication; portfolio data strictly isolated per user.
- Secrets (DB creds, LLM API keys) sourced from a secret manager, never committed.
- Input validation on all writes; protection against injection and prompt
  injection in AI flows.
- Encryption in transit (TLS) and at rest for the database.

### Reliability

- Market-data ingestion failures must not corrupt last-known-good data
  (idempotent, transactional upserts).
- AI provider outages degrade gracefully (dashboard/portfolio remain available;
  insights show a clear "temporarily unavailable" state).

### Availability

- Target **99.5%** availability for core (non-AI) read paths.
- Scheduled jobs retried with backoff; alert on repeated failure.

### Observability

- **OpenTelemetry** traces across API → application → external calls (DB, LLM,
  market-data providers).
- Structured logs with correlation/trace IDs.
- Metrics: request latency, ingestion success rate & freshness, AI call latency &
  token usage/cost, insight generation success rate.
- Every AI interaction is traceable end-to-end (prompt, model, latency, cost,
  outcome) for debuggability and Context Engineering.

### Cost

- The MVP must run at **R$0 / month**: free tiers, free-forever services, or local
  execution only (G12).
- The LLM provider, market-data provider, database host, and app host are each
  swappable via configuration so a free option can be replaced by a paid one
  without code changes.
- AI usage stays within free LLM rate limits via caching and throttling; exceeding
  a free tier must degrade gracefully, never incur charges silently.

---

## 11. Assumptions

- **A1** — MVP is effectively single-user (the author), but the data model and
  auth are multi-user from day one.
- **A2** — Free or low-cost data sources exist for FII quotes and Brazilian macro
  indicators (e.g. BCB/SGS for SELIC/CDI/IPCA, public FII data). Exact providers
  are a blocking decision (§ Dependencies / Risks).
- **A3** — Manual holding entry is acceptable for MVP (no brokerage/B3 import).
- **A4** — "Current estimated value" for fixed income may be approximated from
  `invested_amount`, `annual_rate`, and elapsed time in MVP (not mark-to-market).
- **A5** — Daily market-data freshness is sufficient; intraday is not required.
- **A6** — A **free-tier or local LLM** is sufficient for MVP insight quality
  (e.g. Google Gemini free tier, Groq free tier, or local Ollama). Neither Claude
  nor OpenAI offers a perpetual free tier, so they are deferred to a paid phase —
  but remain drop-in via the `Insighter` port (FR-018, ADR-0003).
- **A7** — Free rate limits (LLM, market data) are adequate for single-user usage;
  insights can be cached/throttled to stay within them.
- **A8** — The entire stack runs at **zero cost** using free tiers, free-forever
  hosts, or local execution (G12, ADR-0003). AWS is a future paid option, not the
  MVP target, because its free tier is a 12-month trial rather than free-forever.

---

## 12. Constraints

### Technical Constraints

- Backend in **Go**; database **PostgreSQL**; frontend **Next.js**.
- AI via a **pluggable LLM provider** behind the `Insighter` port (FR-018). MVP
  uses a **free-tier or local** model (Gemini free tier / Groq / Ollama); Claude
  and OpenAI are paid-phase drop-in upgrades.
- **Docker** for local/dev parity. Deployment target is a **free-forever host**
  (e.g. Oracle Cloud Always Free, Fly.io, Render) with managed free-tier Postgres
  (e.g. Neon / Supabase). **AWS** is a future paid option, not the MVP target.
- Observability via **OpenTelemetry** (free SDK) with a free-tier or self-hosted
  backend (Grafana Cloud free tier / Jaeger / Prometheus).
- Architecture must be **MCP-ready** and **multi-agent-ready** from the start.

### Business Constraints

- **Must never** provide financial advice or specific buy/sell recommendations
  (legal/ethical and product-defining constraint).
- **Zero budget** — the MVP must run entirely on free tiers / free-forever
  services / local execution (G12, ADR-0003).
- Solo developer — scope must stay achievable incrementally.

### Timeline Constraints

- Learning-driven project; phased delivery (§14) prioritized over a fixed deadline.
- Each phase should produce a runnable, demonstrable increment.

---

## 13. Dependencies

### Internal Dependencies

- Investor Profile must exist before goal-aware insights, rebalancing, and health
  score can be meaningful.
- Market Data Module must be ingesting before market-context insights and accurate
  current-value/dashboard figures are possible.

### External Dependencies

- **FII market-data provider** (price, DY, P/VP, sector, last dividend) — must
  have a free access path.
- **Macro data source** — Banco Central do Brasil SGS API for SELIC/CDI/IPCA (free,
  public); an IFIX source.
- **LLM provider** — a free-tier or local model (Gemini free tier / Groq / Ollama),
  behind the `Insighter` port; paid providers (Claude / OpenAI) optional later.
- **Free-forever host + managed free-tier Postgres** (e.g. Oracle Cloud / Fly.io /
  Render + Neon / Supabase). AWS optional in a later paid phase.

---

## 14. Risks

| Risk                                                              | Impact | Mitigation                                                                                       |
| ----------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------ |
| No reliable/free FII data source (price, DY, P/VP, sector)        | High   | Evaluate sources early; abstract behind a `MarketDataProvider` port; allow manual override.       |
| AI output crosses into financial advice                          | High   | Hard guardrail (FR-014): output validation, non-advice prompt design, disclaimer, tests.          |
| Free LLM rate limits / latency too low for usable insights       | Medium | Cache insights per portfolio state; throttle; pick model per task; local Ollama fallback; degrade gracefully (FR-013, Cost NFR). |
| Hallucinated or non-explainable insights                         | High   | Explainability gate (FR-013); ground insights in computed portfolio facts, not free generation.   |
| Free tier discontinued / host's free plan changes                | Medium | Provider behind a port (FR-018) / config; keep ≥1 alternative free option per layer; local Ollama always available. |
| Projection mistaken for a guarantee or for advice                | Medium | Label all projections as estimates; show assumptions per scenario; non-advice disclaimer (FR-014, FR-016/017). |
| Scope creep from rich future vision (agents, MCP)                | Medium | Strict phasing (§ Release Strategy); MVP excludes agents/MCP but architecture stays ready.        |
| Inaccurate fixed-income valuation                                | Medium | Document approximation (A4); refine later with proper mark-to-market.                              |
| Solo-dev bandwidth                                               | Medium | SDD + small, runnable increments; automate tests and observability early.                         |

---

## 15. Release Strategy

### MVP Scope (Phase 1)

A usable, single-user copilot:

- Authentication + Investor Profile (FR-003, FR-015).
- Portfolio Management: FIIs + Fixed Income (FR-001, FR-002).
- Market Data Module: FII + macro ingestion (FR-006, FR-007).
- Dashboard: summary, allocation, sector exposure (FR-004, FR-005).
- AI Insight Engine with explainability + non-advice guardrails (FR-008–FR-010,
  FR-013, FR-014), on a **free-tier/local LLM** behind the `Insighter` port
  (FR-018).
- AI Rebalancing Assistant (FR-011).
- Portfolio Health Score (FR-012).
- Passive Income Projection + Net-Worth Projection (FR-016, FR-017).
- Observability baseline (OpenTelemetry).
- Runs at **zero cost** end-to-end (G12).

### Future Phases

#### Phase 2 — Multi-Agent System

Decompose the Insight Engine into specialized agents orchestrated by a CIO agent:

- **Macro Agent** — SELIC, inflation, economic environment.
- **Fixed Income Agent** — fixed-income products, liquidity, rates.
- **FII Agent** — FII, sector, and dividend analysis.
- **Risk Agent** — concentration, diversification, exposure.
- **CIO Agent** — aggregates agent outputs into a final, explainable report.

#### Phase 3 — MCP Architecture

Expose capabilities as MCP servers/tools so agents (and external clients) consume
them uniformly:

- **Portfolio MCP** — `get_portfolio`, `get_allocations`, `get_dividends`,
  `get_asset_details`.
- **Market MCP** — `get_selic`, `get_ifix`, `get_fii_data`, `get_macro_data`.
- **News MCP** — `get_news`, `get_fii_reports`, `get_market_events`.

#### Phase 4 — Expanded Coverage & Intelligence

- Stocks and ETFs as fully managed asset classes.
- News-driven insights and market events.
- Brokerage/B3 import; richer fixed-income mark-to-market.
- Historical tracking of the Health Score and portfolio over time.

---

## 16. Acceptance Criteria (Product-Level)

The product is considered successful when:

- [ ] User can register and monitor investments (FIIs + Fixed Income).
- [ ] Dashboard correctly reflects portfolio allocation and sector exposure.
- [ ] Market data is automatically and reliably updated.
- [ ] AI generates **explainable** insights (no black-box output).
- [ ] AI can identify portfolio imbalances (concentration, sector, allocation).
- [ ] Rebalancing assistant provides contextual guidance with **no** specific buy
      recommendation.
- [ ] Portfolio Health Score (0–100) is produced with a detailed explanation.
- [ ] Passive income is projected across pessimistic/base/optimistic scenarios.
- [ ] Net worth is projected over time from a configurable monthly contribution.
- [ ] The whole system runs at **zero cost** (free tiers / free-forever / local).
- [ ] The LLM provider is swappable without domain/application changes.
- [ ] Architecture supports future multi-agent expansion (no major redesign).
- [ ] MCP integrations can be added without major redesign.
- [ ] Success metrics in §4 are met.

---

## 17. Future Vision

### Future Enhancements

- Multi-agent CIO reports with cross-agent reasoning.
- MCP ecosystem enabling reuse of portfolio/market/news tools by any AI client.
- News & sentiment signals feeding insights.
- Goal projection / Monte Carlo simulation for retirement and passive-income goals.
- Historical health-score trends and portfolio evolution analytics.
- Optional brokerage/B3 import to eliminate manual entry.

### Long-Term Vision (1–3 years)

YieldForge becomes a trustworthy, explainable Investment Copilot for Brazilian
retail investors — a system that reasons about a full portfolio across asset
classes and macro conditions, surfaces risks and opportunities in plain language,
and empowers better self-directed decisions **without ever crossing into financial
advice**. Technically, it stands as a reference implementation of Go + AI
Engineering + multi-agent + MCP + cloud architecture built with Spec-Driven
Development.
