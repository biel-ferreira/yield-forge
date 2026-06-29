package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// stubInsighter returns a marked insight per call (so we can prove the engine's output comes
// only from the Insighter), and can fail specific calls to exercise degradation.
type stubInsighter struct {
	calls    int
	failWith error         // when set, every call fails with this
	failCall map[int]error // optional: fail specific 0-based call indexes
}

func (s *stubInsighter) Generate(_ context.Context, _ insight.InsightRequest) (insight.InsightResult, error) {
	i := s.calls
	s.calls++
	if s.failWith != nil {
		return insight.InsightResult{}, s.failWith
	}
	if err, ok := s.failCall[i]; ok {
		return insight.InsightResult{}, err
	}
	return insight.InsightResult{
		Insights:   []insight.Insight{{Title: "STUB", Explanation: "from the insighter"}},
		Disclaimer: insight.Disclaimer,
	}, nil
}

func engineWith(t *testing.T, d dashboard.Dashboard, ins insight.Insighter) *Service {
	t.Helper()
	fb := NewFactBuilder(
		fakeDashboard{d: d},
		fakeProfile{err: profile.ErrProfileNotFound},
		fakeMacro{vals: map[marketdata.Indicator]int64{marketdata.IndicatorSELIC: 10_500}},
	)
	return NewService(fb, ins)
}

func TestEngine_Insights_AggregatesAndTags(t *testing.T) {
	stub := &stubInsighter{}
	got, err := engineWith(t, populatedDashboard(), stub).Insights(context.Background(), "u1")
	require.NoError(t, err)

	require.True(t, got.Available)
	require.Equal(t, len(AllCategories), stub.calls, "one Insighter call per category")
	require.Len(t, got.Items, len(AllCategories))
	require.Equal(t, insight.Disclaimer, got.Disclaimer)

	// Every item came from the Insighter (the STUB marker) and is tagged by category.
	cats := map[string]bool{}
	for _, in := range got.Items {
		require.Equal(t, "STUB", in.Title, "AI text comes only from the Insighter")
		cats[in.Category] = true
	}
	require.Equal(t, map[string]bool{"portfolio": true, "allocation": true, "market_context": true}, cats)
}

func TestEngine_Insights_EmptyPortfolioNoLLMCall(t *testing.T) {
	stub := &stubInsighter{}
	empty := dashboard.Dashboard{Allocation: []dashboard.ClassSlice{{Class: dashboard.ClassFII}}}
	got, err := engineWith(t, empty, stub).Insights(context.Background(), "u1")
	require.NoError(t, err)
	require.True(t, got.Available, "available, just nothing to analyse")
	require.Empty(t, got.Items)
	require.Zero(t, stub.calls, "no LLM call for an empty portfolio")
}

func TestEngine_Insights_FullyUnavailable(t *testing.T) {
	stub := &stubInsighter{failWith: insight.ErrInsightsUnavailable}
	got, err := engineWith(t, populatedDashboard(), stub).Insights(context.Background(), "u1")
	require.NoError(t, err, "an LLM outage is degradation, not a hard error")
	require.False(t, got.Available)
	require.Empty(t, got.Items)
}

func TestEngine_Insights_PartialSuccess(t *testing.T) {
	// First category fails (e.g. gate-rejected an order); the others succeed.
	stub := &stubInsighter{failCall: map[int]error{0: insight.ErrAdviceDetected}}
	got, err := engineWith(t, populatedDashboard(), stub).Insights(context.Background(), "u1")
	require.NoError(t, err)
	require.True(t, got.Available, "partial result is still available")
	require.Len(t, got.Items, len(AllCategories)-1, "the rejected category contributes nothing")
}
