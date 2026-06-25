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
- **Feature (`PLAN-1xx`)** — user-facing capabilities.

## Plans

| Plan ID  | Spec     | Title                                    | Status |
| -------- | -------- | ---------------------------------------- | ------ |
| PLAN-001 | SPEC-001 | [Project Scaffolding & Hexagonal Layering](PLAN-001-project-scaffolding-and-layering.md) | ✅ Done |
| PLAN-002 | SPEC-002 | [Persistence Baseline & Migrations](PLAN-002-persistence-baseline-and-migrations.md) | ✅ Done |
| PLAN-003 | SPEC-003 | [Authentication & Per-User Isolation](PLAN-003-authentication-and-per-user-isolation.md) | ✅ Done |
| PLAN-004 | SPEC-004 | [Observability Baseline (OpenTelemetry)](PLAN-004-observability-baseline.md) | ✅ Done |
| PLAN-005 | SPEC-005 | [`Insighter` Port & Free/Local LLM Adapter](PLAN-005-insighter-port-and-llm-adapter.md) | ✅ Done |
| PLAN-006 | SPEC-006 | [`MarketDataProvider` Port & Ingestion Worker](PLAN-006-marketdata-port-and-ingestion-worker.md) | ✅ Done |
| PLAN-101 | SPEC-101 | [Investor Profile](PLAN-101-investor-profile.md) | 📋 Approved |

(Plans are authored just-in-time, one per spec, in the build order below.)

## Standard Phase Order (per the template)

1. Domain Layer → 2. Persistence Layer → 3. Application Layer → 4. API Layer →
5. Observability → 6. Testing → 7. Documentation.

> Plans are authored after their SPEC is approved. Follow the build order in the
> [specs index](../02-specs/README.md#recommended-build-order): foundational plans
> (0xx) before the feature plans (1xx) that depend on them.
