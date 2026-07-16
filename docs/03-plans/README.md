# Implementation Plans (PLANs)

One PLAN per SPEC, sharing the spec's number (and tier). Each defines *how* the
feature is built using the [plan template](../../templates/plan-template.md):
phases, risks, rollout/rollback, and Definition of Done.

## Naming

`PLAN-0NN-short-name.md` (foundational) / `PLAN-1NN-short-name.md` (feature) —
matches its spec, e.g. `PLAN-005-insighter-port.md`,
`PLAN-102-portfolio-management.md`.

## Two-Tier Structure

Mirrors the specs in [`../02-specs/`](../02-specs/):

- **Foundational (`PLAN-0xx`)** — cross-cutting groundwork (scaffolding,
  persistence, auth, ports, observability), planned and built first.
- **Feature (`PLAN-1xx`)** — user-facing backend capabilities.
- **Frontend (`PLAN-2xx`)** — the Next.js web client (foundational `PLAN-20x`,
  feature `PLAN-21x`); mirrors the [`SPEC-2xx` tier](../02-specs/README.md#frontend-specs-spec-2xx).

## Plans

| Plan ID  | Spec     | Title                                    | Status |
| -------- | -------- | ---------------------------------------- | ------ |
| PLAN-001 | SPEC-001 | [Project Scaffolding & Hexagonal Layering](PLAN-001-project-scaffolding-and-layering.md) | ✅ Done |
| PLAN-002 | SPEC-002 | [Persistence Baseline & Migrations](PLAN-002-persistence-baseline-and-migrations.md) | ✅ Done |
| PLAN-003 | SPEC-003 | [Authentication & Per-User Isolation](PLAN-003-authentication-and-per-user-isolation.md) | ✅ Done |
| PLAN-004 | SPEC-004 | [Observability Baseline (OpenTelemetry)](PLAN-004-observability-baseline.md) | ✅ Done |
| PLAN-005 | SPEC-005 | [`Insighter` Port & Free/Local LLM Adapter](PLAN-005-insighter-port-and-llm-adapter.md) | ✅ Done |
| PLAN-006 | SPEC-006 | [`MarketDataProvider` Port & Ingestion Worker](PLAN-006-marketdata-port-and-ingestion-worker.md) | ✅ Done |
| PLAN-007 | SPEC-007 | [Holdings-Driven FII Ticker Ingestion](PLAN-007-holdings-driven-ticker-ingestion.md) | 📝 Draft |
| PLAN-101 | SPEC-101 | [Investor Profile](PLAN-101-investor-profile.md) | ✅ Done |
| PLAN-102 | SPEC-102 | [Portfolio Management (FII + FI)](PLAN-102-portfolio-management.md) | ✅ Done |
| PLAN-103 | SPEC-103 | [Dashboard](PLAN-103-dashboard.md) | ✅ Done |
| PLAN-104 | SPEC-104 | [AI Insight Engine](PLAN-104-ai-insight-engine.md) | ✅ Done |
| PLAN-105 | SPEC-105 | [AI Rebalancing Assistant](PLAN-105-ai-rebalancing-assistant.md) | ✅ Done |
| PLAN-106 | SPEC-106 | [Portfolio Health Score](PLAN-106-portfolio-health-score.md) | ✅ Done |

(Plans are authored just-in-time, one per spec, in the build order below.)

### Pending (spec approved/drafted, plan not yet authored)

| Plan ID  | Spec     | Title                            | Status      |
| -------- | -------- | -------------------------------- | ----------- |
| PLAN-104 | SPEC-104 | AI Insight Engine                | Done |
| PLAN-105 | SPEC-105 | AI Rebalancing Assistant         | Done |
| PLAN-106 | SPEC-106 | Portfolio Health Score           | Done |
| PLAN-107 | SPEC-107 | [Projections (Income & Net Worth)](PLAN-107-projections.md) | ✅ Done |
| PLAN-108 | SPEC-108 | [Conversational Copilot (Chat)](PLAN-108-conversational-copilot.md) | ✅ Done |
| PLAN-109 | SPEC-109 | [Fixed-Income Rate Indexers](PLAN-109-fixed-income-rate-indexers.md) | ✅ Done |

### Frontend (`PLAN-2xx`)

| Plan ID  | Spec     | Title                            | Status      |
| -------- | -------- | -------------------------------- | ----------- |
| PLAN-200 | SPEC-200 | [Frontend App Foundation](PLAN-200-app-foundation.md) | ✅ Done |
| PLAN-210 | SPEC-210 | [Investor Profile Screen](PLAN-210-investor-profile-screen.md) | ✅ Done |
| PLAN-211 | SPEC-211 | [Portfolio Management Screens](PLAN-211-portfolio-management-screens.md) | ✅ Done |
| PLAN-212 | SPEC-212 | [Dashboard Screen (Painel)](PLAN-212-dashboard-screen.md) | ✅ Done |

## Standard Phase Order (per the template)

1. Domain Layer → 2. Persistence Layer → 3. Application Layer → 4. API Layer →
5. Observability → 6. Testing → 7. Documentation.

> Plans are authored after their SPEC is approved. Follow the build order in the
> [specs index](../02-specs/README.md#recommended-build-order): foundational plans
> (0xx) before the feature plans (1xx) that depend on them.
