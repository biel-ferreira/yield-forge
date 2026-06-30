package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// stubInsighter returns a configured narrative result (or error) for the narrative call.
type stubInsighter struct {
	result insight.InsightResult
	err    error
	calls  int
}

func (s *stubInsighter) Generate(_ context.Context, _ insight.InsightRequest) (insight.InsightResult, error) {
	s.calls++
	return s.result, s.err
}

func okNarrative() *stubInsighter {
	return &stubInsighter{result: insight.InsightResult{
		Insights:   []insight.Insight{{Detail: "Sua carteira está saudável.", Explanation: "porque ..."}},
		Disclaimer: insight.Disclaimer,
	}}
}

type fakeDashboard struct {
	d   dashboard.Dashboard
	err error
}

func (f fakeDashboard) GetDashboard(context.Context, string) (dashboard.Dashboard, error) {
	return f.d, f.err
}

type fakeProfile struct {
	p   profile.Profile
	err error
}

func (f fakeProfile) GetProfile(context.Context, string) (profile.Profile, error) {
	return f.p, f.err
}

type fakeHoldings struct {
	h   portfolio.Holdings
	err error
}

func (f fakeHoldings) ListHoldings(context.Context, string) (portfolio.Holdings, error) {
	return f.h, f.err
}

type fakeMacro struct {
	val   int64
	found bool
}

func (f fakeMacro) GetLatestMacroIndicator(_ context.Context, _ marketdata.Indicator) (marketdata.MacroIndicator, error) {
	if !f.found {
		return marketdata.MacroIndicator{}, marketdata.ErrMacroNotFound
	}
	return marketdata.MacroIndicator{Indicator: marketdata.IndicatorSELIC, Value: f.val, Unit: marketdata.UnitBps}, nil
}

func sampleDashboard() dashboard.Dashboard {
	return dashboard.Dashboard{
		Summary: dashboard.Summary{CurrentValueCentavos: 100_000},
		Allocation: []dashboard.ClassSlice{
			{Class: dashboard.ClassFII, ValueCentavos: 60_000, ShareBps: 6000},
			{Class: dashboard.ClassFixedIncome, ValueCentavos: 40_000, ShareBps: 4000},
		},
		FIISectors: []dashboard.SectorSlice{
			{Sector: marketdata.SectorLogistics, ValueCentavos: 40_000, ShareBps: 6667},
			{Sector: marketdata.SectorHybrid, ValueCentavos: 20_000, ShareBps: 3333},
		},
	}
}

func sampleHoldings() portfolio.Holdings {
	return portfolio.Holdings{
		FII: []portfolio.FIIHolding{{Ticker: marketdata.MustParseTicker("HGLG11")}},
		FixedIncome: []portfolio.FixedIncomeHolding{
			{InvestedAmountCentavos: 30_000, LiquidityType: portfolio.LiquidityDaily},
			{InvestedAmountCentavos: 10_000, LiquidityType: portfolio.LiquidityAtMaturity},
		},
	}
}

func newService(d dashboard.Dashboard, h portfolio.Holdings, p fakeProfile, m fakeMacro) *Service {
	return NewService(fakeDashboard{d: d}, p, fakeHoldings{h: h}, m, okNarrative())
}

func newServiceWith(d dashboard.Dashboard, h portfolio.Holdings, p fakeProfile, m fakeMacro, ins insight.Insighter) *Service {
	return NewService(fakeDashboard{d: d}, p, fakeHoldings{h: h}, m, ins)
}

func TestService_BuildInputs_DerivesFacts(t *testing.T) {
	svc := newService(sampleDashboard(), sampleHoldings(),
		fakeProfile{p: profile.Profile{Risk: profile.RiskModerate}}, fakeMacro{val: 1050, found: true})

	in, err := svc.buildInputs(context.Background(), "u1")
	require.NoError(t, err)

	require.Equal(t, int64(100_000), in.CurrentValueCentavos)
	require.Equal(t, int64(60_000), in.FIIValueCentavos)
	require.Equal(t, int64(40_000), in.FixedIncomeValueCentavos)
	require.Equal(t, 3, in.HoldingsCount, "1 FII + 2 fixed income")
	require.Equal(t, 2, in.FIISectorCount)
	require.Equal(t, 6667, in.LargestSectorBps)
	require.Equal(t, 6000, in.LargestClassBps)
	// Liquid = FII (60k) + daily share of FI value: 30k/40k of 40k = 30k → 90k.
	require.Equal(t, int64(90_000), in.LiquidValueCentavos)
	require.True(t, in.HasProfile)
	require.Equal(t, profile.RiskModerate, in.Risk)
	require.True(t, in.HasMacro)
	require.Equal(t, 1050, in.SelicBps)
}

func TestService_BuildInputs_DegradesGracefully(t *testing.T) {
	svc := newService(sampleDashboard(), sampleHoldings(),
		fakeProfile{err: profile.ErrProfileNotFound}, fakeMacro{found: false})

	in, err := svc.buildInputs(context.Background(), "u1")
	require.NoError(t, err, "a not-set profile and missing macro are not errors")
	require.False(t, in.HasProfile)
	require.False(t, in.HasMacro)
}

func TestService_Score_DeterministicAndInRange(t *testing.T) {
	svc := newService(sampleDashboard(), sampleHoldings(),
		fakeProfile{p: profile.Profile{Risk: profile.RiskModerate}}, fakeMacro{val: 1050, found: true})

	a, err := svc.Score(context.Background(), "u1")
	require.NoError(t, err)
	b, err := svc.Score(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, a, b, "same inputs → same score + identical breakdown")
	require.GreaterOrEqual(t, a.Score, 0)
	require.LessOrEqual(t, a.Score, 100)
	require.Len(t, a.Factors, 5)
}

func TestService_Score_AttachesGatedNarrative(t *testing.T) {
	svc := newService(sampleDashboard(), sampleHoldings(),
		fakeProfile{p: profile.Profile{Risk: profile.RiskModerate}}, fakeMacro{val: 1050, found: true})
	hs, err := svc.Score(context.Background(), "u1")
	require.NoError(t, err)
	require.True(t, hs.NarrativeAvailable)
	require.NotEmpty(t, hs.Narrative)
	require.Equal(t, insight.Disclaimer, hs.Disclaimer)
}

func TestService_Score_NumberUnchangedByNarrative(t *testing.T) {
	// THE binding guarantee (D1): the LLM never touches the number. The score is identical whether
	// the narrative succeeds or the LLM is fully down.
	prof := fakeProfile{p: profile.Profile{Risk: profile.RiskModerate}}
	macro := fakeMacro{val: 1050, found: true}

	on, err := newServiceWith(sampleDashboard(), sampleHoldings(), prof, macro, okNarrative()).Score(context.Background(), "u1")
	require.NoError(t, err)
	off, err := newServiceWith(sampleDashboard(), sampleHoldings(), prof, macro,
		&stubInsighter{err: insight.ErrInsightsUnavailable}).Score(context.Background(), "u1")
	require.NoError(t, err, "an LLM outage is not a hard error — the score stands")

	require.Equal(t, off.Score, on.Score, "the narrative never changes the number")
	require.Equal(t, off.Factors, on.Factors, "nor the breakdown")
	require.True(t, on.NarrativeAvailable)
	require.False(t, off.NarrativeAvailable)
	require.Empty(t, off.Narrative)
}

func TestService_Score_EmptyPortfolioNoLLMCall(t *testing.T) {
	stub := okNarrative()
	hs, err := newServiceWith(dashboard.Dashboard{}, portfolio.Holdings{},
		fakeProfile{err: profile.ErrProfileNotFound}, fakeMacro{}, stub).Score(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, 0, hs.Score)
	require.Zero(t, stub.calls, "no LLM call for an empty portfolio")
}
