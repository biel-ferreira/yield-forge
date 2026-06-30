// Package health is the Portfolio Health Score (SPEC-106): a reproducible 0–100 score with a
// per-factor breakdown — diversification, concentration, liquidity, goal alignment, risk exposure.
//
// Unlike the Insight Engine (SPEC-104) and Rebalancing Assistant (SPEC-105), the SCORE and its
// structured breakdown are COMPUTED, never LLM-generated: the PRD reproducibility metric ("same
// inputs → same score + identical explanation") and the binding rule "the LLM never invents
// numbers" both demand it (BR-1062). The score is market-aware — macro (SELIC) is an INPUT to the
// goal-alignment and risk-exposure factors via a modest, documented tilt — so it adjusts with
// conditions yet stays reproducible given (portfolio, profile, macro). An optional gated Insighter
// narrative (the "professor") explains the computed result using the live market; it is grounded,
// gated (FR-013/014), degradable, and NEVER changes the number.
//
// The Compute core is pure (no SQL/HTTP/LLM/time); the service composes the dashboard (SPEC-103),
// profile (SPEC-101), holdings (SPEC-102), and macro (SPEC-006) reads at the edge.
package health
