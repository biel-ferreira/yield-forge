package rebalancing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// fakeFactSource returns canned base facts (the published BuildFacts seam).
type fakeFactSource struct {
	facts insight.Facts
	err   error
}

func (f fakeFactSource) BuildFacts(context.Context, string) (insight.Facts, error) {
	// Return a fresh copy so the engine's in-place augmentation never leaks across calls.
	out := make(insight.Facts, len(f.facts))
	for k, v := range f.facts {
		out[k] = v
	}
	return out, f.err
}

// fakeUniverse returns a canned FII universe.
type fakeUniverse struct {
	quotes []marketdata.FIIQuote
	err    error
}

func (f fakeUniverse) ListFIIUniverse(context.Context) ([]marketdata.FIIQuote, error) {
	return f.quotes, f.err
}

// scriptedInsighter returns a configured result per call, in order (so we can drive areas then
// candidates), and can fail specific calls to exercise degradation.
type scriptedInsighter struct {
	calls    int
	results  []insight.InsightResult
	failWith error
}

func (s *scriptedInsighter) Generate(_ context.Context, _ insight.InsightRequest) (insight.InsightResult, error) {
	i := s.calls
	s.calls++
	if s.failWith != nil {
		return insight.InsightResult{}, s.failWith
	}
	if i < len(s.results) {
		return s.results[i], nil
	}
	return insight.InsightResult{}, nil
}

func areaExplanation(title string) insight.InsightResult {
	return insight.InsightResult{
		Insights:   []insight.Insight{{Title: title, Detail: "...", Explanation: "porque ..."}},
		Disclaimer: insight.Disclaimer,
	}
}

func engineFacts() insight.Facts {
	return insight.Facts{
		"current_value_centavos": int64(100_000),
		"allocation_bps":         map[string]int{"fii": 10000},
		"risk_profile":           "moderate",
	}
}

func universeHGLG() []marketdata.FIIQuote {
	return []marketdata.FIIQuote{
		{Ticker: marketdata.MustParseTicker("HGLG11"), Sector: marketdata.SectorLogistics, DividendYieldBps: 850},
	}
}

func TestRebalance_AreasCarryComputedShareAndExplanation(t *testing.T) {
	// 100% FII + moderate → split steers to Fixed Income (one area) → one area call + one
	// candidates call.
	ins := &scriptedInsighter{results: []insight.InsightResult{
		areaExplanation("Renda Fixa"), // area call
		{},                            // candidates call (none)
	}}
	svc := NewService(fakeFactSource{facts: engineFacts()}, fakeUniverse{quotes: universeHGLG()}, ins)

	got, err := svc.Rebalance(context.Background(), "u1", mustContribution(t, 100_000), Options{})
	require.NoError(t, err)
	require.True(t, got.Available)
	require.Len(t, got.Areas, 1)
	require.Equal(t, "fixed_income", got.Areas[0].Class)
	require.Equal(t, 10000, got.Areas[0].SuggestedShareBps, "computed share, not from the LLM")
	require.NotEmpty(t, got.Areas[0].Explanation, "every area carries an explanation (FR-013)")
	require.Equal(t, insight.Disclaimer, got.Disclaimer)
}

func TestRebalance_GroundingGuardDropsUnknownTicker(t *testing.T) {
	// The candidates call names one known (HGLG11) and one hallucinated (FAKE11) ticker.
	ins := &scriptedInsighter{results: []insight.InsightResult{
		areaExplanation("Renda Fixa"), // area call
		{Insights: []insight.Insight{ // candidates call
			{Title: "HGLG11", Explanation: "logística sólida"},
			{Title: "FAKE11", Explanation: "inventado"},
		}},
	}}
	svc := NewService(fakeFactSource{facts: engineFacts()}, fakeUniverse{quotes: universeHGLG()}, ins)

	got, err := svc.Rebalance(context.Background(), "u1", mustContribution(t, 100_000), Options{})
	require.NoError(t, err)
	require.Len(t, got.Candidates, 1, "the hallucinated ticker is dropped (grounding guard)")
	require.Equal(t, "HGLG11", got.Candidates[0].Ticker)
	require.Equal(t, "logistics", got.Candidates[0].Sector)
	require.Zero(t, got.Candidates[0].IllustrativeShareBps, "no per-asset share unless opted in")
}

func TestRebalance_IncludeAssetSharesSplitsFIIArea(t *testing.T) {
	// Two known candidates + an FII area present → the FII area's bps is split across them.
	uni := []marketdata.FIIQuote{
		{Ticker: marketdata.MustParseTicker("HGLG11"), Sector: marketdata.SectorLogistics},
		{Ticker: marketdata.MustParseTicker("KNRI11"), Sector: marketdata.SectorHybrid},
	}
	// Empty portfolio + aggressive → FII area present (70%).
	facts := insight.Facts{"current_value_centavos": int64(0), "risk_profile": "aggressive"}
	ins := &scriptedInsighter{results: []insight.InsightResult{
		areaExplanation("FIIs"),       // area: fii
		areaExplanation("Renda Fixa"), // area: fixed_income
		{Insights: []insight.Insight{ // candidates
			{Title: "HGLG11", Explanation: "a"},
			{Title: "KNRI11", Explanation: "b"},
		}},
	}}
	svc := NewService(fakeFactSource{facts: facts}, fakeUniverse{quotes: uni}, ins)

	got, err := svc.Rebalance(context.Background(), "u1", mustContribution(t, 1_000_000), Options{IncludeAssetShares: true})
	require.NoError(t, err)
	require.Len(t, got.Candidates, 2)
	// FII area is 7000 bps → 3500 each (illustrative).
	require.Equal(t, 3500, got.Candidates[0].IllustrativeShareBps)
	require.Equal(t, 3500, got.Candidates[1].IllustrativeShareBps)
}

func TestRebalance_FullyUnavailable(t *testing.T) {
	ins := &scriptedInsighter{failWith: insight.ErrInsightsUnavailable}
	svc := NewService(fakeFactSource{facts: engineFacts()}, fakeUniverse{quotes: universeHGLG()}, ins)

	got, err := svc.Rebalance(context.Background(), "u1", mustContribution(t, 100_000), Options{})
	require.NoError(t, err, "an LLM outage is degradation, not a hard error")
	require.False(t, got.Available)
	require.Empty(t, got.Areas)
}

func TestRebalance_EmptyPortfolioStillGuides(t *testing.T) {
	// No holdings + a contribution → still produces area guidance (unlike SPEC-104 insights).
	facts := insight.Facts{"current_value_centavos": int64(0), "risk_profile": "moderate"}
	ins := &scriptedInsighter{results: []insight.InsightResult{
		areaExplanation("FIIs"), areaExplanation("Renda Fixa"), {},
	}}
	svc := NewService(fakeFactSource{facts: facts}, fakeUniverse{}, ins)

	got, err := svc.Rebalance(context.Background(), "u1", mustContribution(t, 50_000), Options{})
	require.NoError(t, err)
	require.True(t, got.Available)
	require.NotEmpty(t, got.Areas, "empty portfolio + contribution still guides")
}
