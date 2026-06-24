package insight

import "context"

// Fake is a deterministic Insighter for tests, CI, and the "AI disabled" mode
// (INSIGHTER_PROVIDER=fake) — no model, no network, no cost (SPEC-005 FR-509). Its
// output is explainable and order-free, so it passes the gate. The disclaimer is
// attached by the gate, not here.
type Fake struct{}

// Compile-time check that Fake satisfies the port.
var _ Insighter = Fake{}

// Generate returns a fixed insight, or ErrInsufficientFacts when no facts are given
// (so callers exercise the same empty-facts path as the real adapters).
func (Fake) Generate(_ context.Context, req InsightRequest) (InsightResult, error) {
	if len(req.Facts) == 0 {
		return InsightResult{}, ErrInsufficientFacts
	}
	return InsightResult{
		Insights: []Insight{{
			Category:    "exemplo",
			Title:       "Análise de exemplo",
			Detail:      "Esta é uma observação de exemplo, gerada sem um modelo de IA.",
			Explanation: "Saída determinística do Insighter de teste, usada em CI e quando a IA está desativada.",
		}},
	}, nil
}
