# Feature Specifications (SPECs)

One SPEC per coherent capability — never a single mega-document. Each translates a
slice of the [PRD](../01-product/PRD.md) into concrete, buildable behavior using the
[spec template](../../templates/spec-template.md), and has its own acceptance
criteria and Definition of Done.

## Two-Tier Structure

Specs are split into two tiers so cross-cutting groundwork is explicit and ordered
ahead of the user-facing features that depend on it (see PRD §13 Dependencies).

- **Foundational (`SPEC-0xx`)** — cross-cutting seams and infrastructure with no
  user-facing screen of their own, but which many features depend on (project
  skeleton, persistence, auth, the `Insighter` / `MarketDataProvider` ports,
  observability). Built first.
- **Feature (`SPEC-1xx`)** — user-facing capabilities that deliver product value.

## Naming

`SPEC-0NN-short-name.md` (foundational) / `SPEC-1NN-short-name.md` (feature) —
e.g. `SPEC-005-insighter-port.md`, `SPEC-102-portfolio-management.md`.

---

## Foundational Specs (`SPEC-0xx`)

| Spec ID  | Capability                              | PRD refs                | Status      |
| -------- | --------------------------------------- | ----------------------- | ----------- |
| SPEC-001 | [Project Scaffolding & Hexagonal Layering](SPEC-001-project-scaffolding-and-layering.md) | ADR-0002, §10 NFR | ✅ Done |
| SPEC-002 | [Persistence Baseline & Migrations](SPEC-002-persistence-baseline-and-migrations.md) | §7 Data, NFR | ✅ Done |
| SPEC-003 | [Authentication & Per-User Isolation](SPEC-003-authentication-and-per-user-isolation.md) | FR-015 | ✅ Done |
| SPEC-004 | [Observability Baseline (OpenTelemetry)](SPEC-004-observability-baseline.md) | §10 Observability | ✅ Done |
| SPEC-005 | [`Insighter` Port & Free/Local LLM Adapter](SPEC-005-insighter-port-and-llm-adapter.md) | FR-018, FR-013, FR-014 | ✅ Done |
| SPEC-006 | [`MarketDataProvider` Port & Ingestion Worker](SPEC-006-marketdata-port-and-ingestion-worker.md) | FR-006, FR-007 | ✅ Done |

> SPEC-005 defines the explainability (FR-013) and non-advice (FR-014) gates as
> middleware wrapping the port, so every AI feature inherits them (ADR-0002/0003).

## Feature Specs (`SPEC-1xx`)

| Spec ID  | Feature                          | PRD FRs               | Status      |
| -------- | -------------------------------- | --------------------- | ----------- |
| SPEC-101 | [Investor Profile](SPEC-101-investor-profile.md) | FR-003                | ✅ Done |
| SPEC-102 | Portfolio Management (FII + FI)  | FR-001, FR-002        | Not started |
| SPEC-103 | Dashboard                        | FR-004, FR-005        | Not started |
| SPEC-104 | AI Insight Engine                | FR-008–FR-010 (+ FR-013/014 via SPEC-005) | Not started |
| SPEC-105 | AI Rebalancing Assistant         | FR-011, FR-014        | Not started |
| SPEC-106 | Portfolio Health Score           | FR-012                | Not started |
| SPEC-107 | Projections (Income & Net Worth) | FR-016, FR-017        | Not started |

Every PRD functional requirement (FR-001…FR-018) maps to exactly one owning spec
above.

---

## Recommended Build Order

Foundations first, then features in dependency order:

```
SPEC-001 → 002 → 003 → 004        (skeleton, DB, auth, observability)
        ↓
SPEC-005, SPEC-006                (ports: LLM + market data — build in parallel)
        ↓
SPEC-101 Profile → SPEC-102 Portfolio
        ↓
SPEC-103 Dashboard                (needs 102 + 006)
        ↓
SPEC-104 Insights → 105 Rebalancing → 106 Health Score   (need 005 + 101/102/006)
        ↓
SPEC-107 Projections              (needs 102 + 006; deterministic, no LLM)
```

> A feature is not built until it has a SPEC, and a SPEC is not built until it has a
> matching PLAN (same number) in [`../03-plans/`](../03-plans/).
