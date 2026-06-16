# ADR-0001 — Record Architecture Decisions

| Field   | Value      |
| ------- | ---------- |
| Status  | Accepted   |
| Date    | 2026-06-16 |
| Deciders| Gabigol    |

## Context

YieldForge is built with Spec-Driven Development and is intended to grow from a
single-LLM MVP into a multi-agent, MCP-based system. Significant technical
decisions (frameworks, providers, runtime, AI strategy) will be made over time. To
keep the SDD documents trustworthy and to make the reasoning behind the
architecture auditable, decisions need to be recorded where they can be reviewed
and superseded — not lost in chat history or commit messages.

## Decision

We will record every significant architectural decision as an **Architecture
Decision Record (ADR)** under [`docs/04-architecture/adr/`](.), using the
lightweight **Context · Decision · Consequences** format.

- ADRs are numbered sequentially and zero-padded (`ADR-0001`, `ADR-0002`, …).
- An accepted ADR is **immutable**; changing a decision means writing a new ADR
  that marks the old one `Superseded by ADR-XXXX`.
- The ADR index lives in [`adr/README.md`](README.md).

## Consequences

- **Positive:** decisions and their rationale are discoverable, reviewable, and
  reversible-by-supersession; new contributors (or future-self) understand *why*.
- **Positive:** SPECs and PLANs can reference ADRs instead of re-arguing settled
  points.
- **Cost:** a small amount of discipline/overhead per significant decision.
