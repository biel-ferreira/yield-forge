# YieldForge — Spec-Driven Development (SDD) Workspace

This folder is the single source of truth for **what** we are building and **why**,
following a Spec-Driven Development workflow. Code is written to satisfy these
documents — not the other way around.

> **Product:** YieldForge — an AI-powered Investment Copilot for Brazilian retail
> investors (FIIs and Fixed Income first).
> **Status:** Pre-development (defining the PRD).

---

## The SDD Flow

```
        ┌─────────┐      ┌──────────┐      ┌──────────┐      ┌──────┐
 IDEA → │   PRD   │  →   │   SPEC   │  →   │   PLAN   │  →   │ CODE │
        └─────────┘      └──────────┘      └──────────┘      └──────┘
         What & Why       How (per         How to build       Implementation
         (product)        feature)         (per spec)
```

1. **PRD** (Product Requirements Document) — one per product. Defines vision,
   scope, personas, user stories, functional & non-functional requirements,
   success criteria, and the release roadmap. → [`01-product/PRD.md`](01-product/PRD.md)

2. **SPEC** (Feature Specification) — one per feature. Translates a slice of the
   PRD into concrete behavior: functional requirements, user flows, business
   rules, domain model, API contracts, data storage, edge cases, observability,
   and testing strategy. → [`02-specs/`](02-specs/)

3. **PLAN** (Implementation Plan) — one per SPEC. Defines the execution strategy:
   phases (domain → persistence → application → API → observability → tests),
   risks, rollout/rollback, and Definition of Done. → [`03-plans/`](03-plans/)

4. **ARCHITECTURE** — cross-cutting system design and the decision log (ADRs)
   that the specs and plans must respect. → [`04-architecture/`](04-architecture/)

---

## Folder Map

| Folder | Purpose |
| ------ | ------- |
| [`01-product/`](01-product/) | Product Requirements Document (PRD). The "north star". |
| [`02-specs/`](02-specs/) | Feature Specifications. One file per feature (`SPEC-XXX-name.md`). |
| [`03-plans/`](03-plans/) | Implementation Plans. One file per spec (`PLAN-XXX-name.md`). |
| [`04-architecture/`](04-architecture/) | System architecture overview + Architecture Decision Records (ADRs). |
| [`../templates/`](../templates/) | The source templates for PRD / SPEC / PLAN. |

---

## Naming Conventions

- **PRD:** `PRD.md` (one per product).
- **Spec:** one file per capability, in two tiers —
  foundational `SPEC-0NN-short-name.md` (e.g. `SPEC-005-insighter-port.md`) and
  feature `SPEC-1NN-short-name.md` (e.g. `SPEC-102-portfolio-management.md`).
  See the [specs index](02-specs/README.md) for the full list and build order.
- **Plan:** `PLAN-<same-number>-short-name.md` — mirrors its spec's number and tier.
- **ADR:** `ADR-0001-short-title.md`. ADRs are immutable once accepted; supersede
  rather than edit.

## Working Agreement

- A feature is not started until it has a **SPEC**; a SPEC is not built until it
  has a **PLAN**.
- Every AI-generated insight in the product must be **explainable** (a core
  product principle — see the PRD). This constraint flows down into every spec.
- The product **assists** decisions and **never** gives buy/sell financial advice.
- Update [`../CHANGELOG.md`](../CHANGELOG.md) (the `[Unreleased]` section) in the
  **same PR** as any notable change — Keep a Changelog format, SemVer headings.
