// Package projection is the Projections feature (SPEC-107): two forward-looking views over the
// current portfolio — a Passive Income Projection (monthly/annual income across pessimistic / base
// / optimistic scenarios, FR-016) and a Net-Worth Projection (value over a configurable horizon
// from current value + reinvested income + a configurable monthly contribution, FR-017).
//
// Like the Dashboard (SPEC-103) and the Health Score core (SPEC-106), it is a PURE deterministic
// computation — FR-016 requires "same inputs → same result", so there is NO LLM: figures are
// computed, never generated (BR-1072). Money is int64 centavos and rates integer basis points,
// half-up throughout (BR-1071); the net-worth series compounds monthly and is emitted as yearly
// points for charting. The scenarios' assumptions are documented, exposed parameters (BR-1073); the
// output is a labelled estimate, never a transaction order, so FR-014 holds by construction.
//
// The Compute core is pure (no SQL/HTTP/LLM/time); the service composes the dashboard (SPEC-103)
// and holdings (SPEC-102) reads at the edge.
package projection
