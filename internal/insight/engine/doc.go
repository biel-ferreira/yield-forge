// Package engine is the AI Insight Engine (SPEC-104): it builds deterministic facts and
// turns them into explainable Portfolio / Allocation / Market-Context insights (FR-008/009/010).
//
// It is the application layer of the insight feature — the pure insight core (the Insighter
// port + gates, SPEC-005) stays in the parent package; this subpackage reads the dashboard
// (SPEC-103), profile (SPEC-101), and macro (SPEC-006) seams and orchestrates the Insighter.
// Two binding rules shape it: facts are COMPUTED, not generated (the deterministic FactBuilder,
// BR-1041), and the guards hold BY CONSTRUCTION — user-facing AI text is emitted ONLY through
// the Insighter, so explainability (FR-013) and non-advice (FR-014) are unavoidable (BR-1042).
//
// The FactBuilder is a published, reusable seam (BuildFacts): the Conversational Copilot
// (SPEC-108) grounds each chat turn through it, and the Rebalancing Assistant / Health Score
// (SPEC-105/106) reuse it. No AI output is constructed outside the Insighter anywhere here.
package engine
