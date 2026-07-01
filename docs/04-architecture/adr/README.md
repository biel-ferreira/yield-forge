# Architecture Decision Records (ADRs)

An ADR captures a single significant architectural decision: its context, the
decision, and its consequences. ADRs are **immutable once accepted** — to change a
decision, write a new ADR that *supersedes* the old one.

## Format

Each ADR uses: **Status · Context · Decision · Consequences**.

## Naming

`ADR-XXXX-short-title.md` — sequential, zero-padded.

## Index

| ADR  | Title                          | Status   |
| ---- | ------------------------------ | -------- |
| 0001 | Record architecture decisions  | Accepted |
| 0002 | Tech stack and backend layering | Accepted |
| 0003 | Zero-cost infra & pluggable LLM | Accepted |
| 0004 | [Frontend repository strategy (mono-repo)](ADR-0004-frontend-repository-strategy.md) | Proposed |
| 0005 | [Conversational copilot orchestration](ADR-0005-conversational-copilot-orchestration.md) | Proposed |
| 0006 | [Frontend UI stack & design system](ADR-0006-frontend-ui-stack-and-design-system.md) | Accepted |

Statuses: `Proposed` · `Accepted` · `Superseded by ADR-XXXX` · `Deprecated`.
