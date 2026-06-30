// Package rebalancing is the AI Rebalancing Assistant (SPEC-105): given a contribution amount, it
// produces explainable guidance on where to focus the new money — suggested allocation areas (each
// with a deterministically computed share of the contribution), optional grounded named candidates,
// and the non-advice disclaimer — never a transaction order (FR-011, FR-014).
//
// It is the second consumer of the published SPEC-104 Fact Builder seam (BuildFacts). Three rules
// shape it: the suggested percentages are COMPUTED, not generated (the deterministic allocator,
// BR-1056) and so are the candidate names (the grounding guard drops any ticker the system does
// not know, BR-1053); the guards hold BY CONSTRUCTION — user-facing text is emitted only through
// the gated Insighter (SPEC-005), so explainability (FR-013) and non-advice (FR-014) are
// unavoidable (BR-1054); and money is integer centavos / basis points that reconcile (the split
// sums to exactly 10 000 bps, half-up). Live market data tilts the LLM's explanation, not the
// computed number (D7).
package rebalancing
