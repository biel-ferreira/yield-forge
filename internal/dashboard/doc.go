// Package dashboard is the Dashboard feature (SPEC-103): it computes the investor's
// portfolio summary — total invested, current estimated value (the full patrimony / net
// worth), monthly passive income, and growth (FR-004) — plus the allocation breakdown by
// asset class and FII sector exposure (FR-005).
//
// It is a read-only COMPUTE feature: it owns no tables and writes nothing. It reads holdings
// (SPEC-102) and FII quotes (SPEC-006) through small consumer-defined ports, and the figures
// are produced by a pure, deterministic function — int64 centavos and integer basis points,
// half-up rounding, no float — so the same inputs always yield the same figures and every
// breakdown reconciles against the underlying holdings (PRD §6, BR-1031/BR-1034). No AI output
// is produced here (FR-013/FR-014 do not apply); these deterministic facts are the substrate
// the Fact Builder (SPEC-104) and the projections (SPEC-107) build on.
package dashboard
