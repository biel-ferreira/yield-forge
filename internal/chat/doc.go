// Package chat is the Conversational Copilot (SPEC-108): a multi-turn, fact-grounded chat where the
// investor asks free-form questions and every reply is grounded in computed facts and emitted ONLY
// through the gated Insighter (SPEC-005) — so explainability (FR-013) and non-advice (FR-014) hold
// turn by turn.
//
// It is the capstone of the AI feature set: it invents no new reasoning engine, it ORCHESTRATES the
// ones already built — insights (SPEC-104), rebalancing (SPEC-105), projections (SPEC-107) — behind
// a chat surface via lightweight intent routing, and is the deliberate bridge to the Phase-2
// multi-agent CIO + Phase-3 MCP (ADR-0005). Three rules shape it: facts are COMPUTED, not generated
// (each turn grounded in a deterministic snapshot; prior assistant text is dialogue context, never a
// source of figures — BR-1081); the guards hold BY CONSTRUCTION (text only via the Insighter —
// BR-1082); and identity comes from the session context, threads/messages/facts scoped per-user
// (BR-1083). Conversation memory is bounded + clearable to stay zero-cost (BR-1085).
//
// The domain (roles, intents, the classifier, entities) is pure; SQL lives in chat/postgres and the
// LLM/read seams are consumed through consumer-defined ports at the edge.
package chat
