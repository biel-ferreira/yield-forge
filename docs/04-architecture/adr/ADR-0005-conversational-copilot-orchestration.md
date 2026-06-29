# ADR-0005 — Conversational Copilot Orchestration (single-LLM tool-light now, multi-agent CIO later)

| Field    | Value      |
| -------- | ---------- |
| Status   | Proposed   |
| Date     | 2026-06-29 |
| Deciders | Gabigol    |
| Related  | [PRD §6, §15](../../01-product/PRD.md), [SPEC-108](../../02-specs/SPEC-108-conversational-copilot.md), [SPEC-104](../../02-specs/SPEC-104-ai-insight-engine.md), [SPEC-005](../../02-specs/SPEC-005-insighter-port-and-llm-adapter.md), [ADR-0002](ADR-0002-tech-stack-and-layering.md), [ADR-0003](ADR-0003-zero-cost-and-pluggable-llm.md) |

## Context

[SPEC-108](../../02-specs/SPEC-108-conversational-copilot.md) introduces a **conversational
copilot** — a multi-turn chat where the investor asks free-form questions about their portfolio,
allocation, market context, and the current month's contribution strategy. This is the first
feature whose primary input is **free-text from the user** rather than a structured request, and the
first that must sustain a **dialogue** across turns.

That raises a genuine architecture decision: **how does a chat turn obtain the facts it reasons
over, and how much agency does the LLM have in fetching them?** The PRD's north star is a Phase 2
**multi-agent CIO** fanning out to specialized agents over **Phase 3 MCP tools** ([PRD §15](../../01-product/PRD.md)),
and a first-class success criterion is reaching it **without major redesign**. The chat is the
surface where that vision becomes visible, so the grounding mechanism we pick now must not paint us
into a corner.

Forces specific to this project:

- **Binding guards must hold turn by turn.** Explainability (FR-013) and non-advice (FR-014) are
  enforced as middleware wrapping the `Insighter` port ([SPEC-005](../../02-specs/SPEC-005-insighter-port-and-llm-adapter.md)).
  A chat invites "devo comprar XPML11?", so the gate is exactly what keeps the answer a reasoned
  consideration, not an order. Any design that emitted AI text *outside* the port would bypass the
  gates — unacceptable.
- **Facts are computed, not generated** ([PRD §6](../../01-product/PRD.md)). The numbers must come
  from the deterministic Fact Builder ([SPEC-104](../../02-specs/SPEC-104-ai-insight-engine.md)),
  never from the model, and never from prior generated text fed back in.
- **Zero cost + free-tier rate limits** ([ADR-0003](ADR-0003-zero-cost-and-pluggable-llm.md)). A
  free/local LLM with tight rate limits cannot afford a multi-round tool-calling loop per turn in the
  MVP; an unbounded conversation context would also blow the cost/latency budget.
- **The `Insighter` port is the seam** ([ADR-0002](ADR-0002-tech-stack-and-layering.md)). It already
  backs `/insights`; a single-LLM call today and a CIO-orchestrated fleet later are both *adapters*
  behind the same port.

## Decision

**Ground each chat turn with a pre-built deterministic fact snapshot and emit the reply only through
the `Insighter` port — keep live agentic tool-calling out of the MVP, behind the same seam.**

Concretely:

1. **Pre-built fact snapshot, not a tool-call loop.** Per turn, the chat engine runs deterministic
   **intent routing** (general vs. "tenho R$X pra aportar") and calls the **Fact Builder** (reused
   from SPEC-104; the rebalancing facts from SPEC-105 for a contribution turn) to assemble one
   `insight.Facts` snapshot. That snapshot + a **bounded window** of prior turns + the new question
   become an `insight.InsightRequest{ Facts, Task: chat, UserID }`. The LLM does **not** iteratively
   request data mid-turn in the MVP.

2. **One path to user-facing text: the `Insighter`.** The chat adds a new `insight.Task` value
   (`chat`) but **no new gate, provider, or bypass** — every assistant message passes the existing
   explainability + non-advice gates and carries the disclaimer. The engine never constructs
   user-facing AI text outside the port.

3. **Numbers never come from generated text.** Prior assistant messages are included only as dialogue
   context; figures always come from the freshly computed snapshot. The conversation store is durable
   convenience, never a source of truth for numbers.

4. **Bounded, cancellable, cheap.** The prior-turn window and the per-user conversation store are
   **bounded** (rolling eviction) to respect free-tier limits; the Insighter's cache/throttle/degrade
   chain (SPEC-005) is reused unchanged.

5. **The seam is the bridge to Phase 2/3.** Because grounding sits behind the same `Insighter` port,
   swapping the single-LLM turn for a **CIO orchestrator** that fans out to agents pulling facts over
   **MCP tools** is an *adapter* change — the chat surface, the gates, and the API contract stay put.
   Live agentic MCP tool-calling is therefore explicitly the **future evolution** of point (1), not a
   throwaway MVP shortcut.

**Alternative considered — agentic tool-calling from day one** (the LLM iteratively calls
`get_portfolio` / `get_selic` MCP tools within a turn): rejected for the MVP. It is the right Phase
2/3 design, but in Phase 1 it multiplies LLM round-trips per turn (breaking free-tier rate/latency
budgets, ADR-0003), adds non-determinism to a flow we want reproducible, and front-loads the MCP
tool layer before any agent exists to use it. The chosen design reaches the same destination behind
the same port, incrementally.

**Alternative considered — feed the whole conversation (and prior answers) back as context**:
rejected. It risks the model treating its own earlier generated numbers as fact (violating
"facts are computed, not generated"), and grows context cost without bound. A bounded prior-turn
window for *dialogue continuity* plus a freshly computed fact snapshot for *numbers* keeps both
properties.

## Consequences

- **Positive:** the binding guards (FR-013/FR-014) hold for the chat by construction — it reuses the
  gated `Insighter`, adding only a `chat` task.
- **Positive:** deterministic and zero-cost — one fact build + one gated LLM call per turn, within
  free-tier limits; bounded memory keeps storage and context cost flat.
- **Positive:** "facts are computed, not generated" is preserved even in a conversational setting.
- **Positive:** the Phase 2 multi-agent CIO and Phase 3 MCP land as adapters behind the existing
  `Insighter` port — the chat surface and API contract do not change (the "no major redesign"
  success criterion).
- **Cost / tradeoff:** the MVP chat cannot *dynamically* fetch a fact the pre-built snapshot didn't
  include (e.g. a one-off detail about a specific ticker the user names). Accepted: intent routing
  covers the common cases, and live tool-calling is the documented next step.
- **Cost / tradeoff:** bounding the prior-turn window means very long conversations lose early
  context; a rolling summary is a deferred Open Question (SPEC-108 §15).
- **Open:** SSE streaming, agentic MCP tool-calling, thread summarisation, and gated title generation
  are deferred to SPEC-108's Open Questions and, where significant, a future ADR superseding the
  tool-calling stance here.
