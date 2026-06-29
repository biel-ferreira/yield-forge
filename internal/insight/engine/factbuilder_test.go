package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

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

type fakeMacro struct {
	vals map[marketdata.Indicator]int64
}

func (f fakeMacro) GetLatestMacroIndicator(_ context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error) {
	v, ok := f.vals[ind]
	if !ok {
		return marketdata.MacroIndicator{}, marketdata.ErrMacroNotFound
	}
	return marketdata.MacroIndicator{Indicator: ind, Value: v, Unit: marketdata.UnitBps}, nil
}

func populatedDashboard() dashboard.Dashboard {
	return dashboard.Dashboard{
		Summary: dashboard.Summary{
			TotalInvestedCentavos: 2_675_000, CurrentValueCentavos: 2_820_000,
			MonthlyIncomeCentavos: 11_000, GrowthCentavos: 145_000, GrowthBps: 542,
		},
		Allocation: []dashboard.ClassSlice{
			{Class: dashboard.ClassFII, ValueCentavos: 1_700_000, ShareBps: 6028},
			{Class: dashboard.ClassFixedIncome, ValueCentavos: 1_120_000, ShareBps: 3972},
			{Class: dashboard.ClassStocks, ValueCentavos: 0, ShareBps: 0},
			{Class: dashboard.ClassETFs, ValueCentavos: 0, ShareBps: 0},
		},
		FIISectors: []dashboard.SectorSlice{
			{Sector: marketdata.SectorLogistics, ValueCentavos: 1_600_000, ShareBps: 9411},
			{Sector: marketdata.SectorOther, ValueCentavos: 100_000, ShareBps: 588},
		},
		StaleTickers: []string{"XPLG11"},
	}
}

func moderateProfile(t *testing.T) profile.Profile {
	t.Helper()
	h, err := profile.ParseHorizon(10)
	require.NoError(t, err)
	return profile.Profile{
		Risk:       profile.RiskModerate,
		Objectives: []profile.Objective{profile.ObjectiveRetirement, profile.ObjectivePassiveIncome},
		Horizon:    h,
	}
}

func TestBuildFacts_Full(t *testing.T) {
	b := NewFactBuilder(
		fakeDashboard{d: populatedDashboard()},
		fakeProfile{p: moderateProfile(t)},
		fakeMacro{vals: map[marketdata.Indicator]int64{
			marketdata.IndicatorSELIC: 10_500, marketdata.IndicatorCDI: 10_250, marketdata.IndicatorIPCA: 393,
		}},
	)

	f, err := b.BuildFacts(context.Background(), "u1")
	require.NoError(t, err)

	require.Equal(t, int64(2_820_000), f["current_value_centavos"])
	require.Equal(t, 542, f["growth_bps"])
	require.Equal(t, map[string]int{"fii": 6028, "fixed_income": 3972}, f["allocation_bps"], "zero-value classes omitted")
	require.Equal(t, map[string]int{"logistics": 9411, "other": 588}, f["fii_sector_bps"])
	require.Equal(t, 9411, f["largest_fii_sector_bps"])
	require.Equal(t, 2, f["fii_sector_count"])
	require.Equal(t, []string{"XPLG11"}, f["stale_tickers"])
	require.Equal(t, "moderate", f["risk_profile"])
	require.Equal(t, []string{"retirement", "passive_income"}, f["objectives"])
	require.Equal(t, 10, f["horizon_years"])
	require.Equal(t, map[string]int64{"selic_bps": 10_500, "cdi_bps": 10_250, "ipca_bps": 393}, f["macro_bps"])
}

func TestBuildFacts_Deterministic(t *testing.T) {
	b := NewFactBuilder(
		fakeDashboard{d: populatedDashboard()},
		fakeProfile{p: moderateProfile(t)},
		fakeMacro{vals: map[marketdata.Indicator]int64{marketdata.IndicatorSELIC: 10_500}},
	)
	a, err := b.BuildFacts(context.Background(), "u1")
	require.NoError(t, err)
	c, err := b.BuildFacts(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, a, c, "same inputs → same facts")
}

func TestBuildFacts_ProfileNotSetAndMissingMacro(t *testing.T) {
	b := NewFactBuilder(
		fakeDashboard{d: populatedDashboard()},
		fakeProfile{err: profile.ErrProfileNotFound},
		fakeMacro{vals: nil}, // all macro missing
	)
	f, err := b.BuildFacts(context.Background(), "u1")
	require.NoError(t, err)
	require.NotContains(t, f, "risk_profile", "a not-set profile is omitted, not an error")
	require.NotContains(t, f, "macro_bps", "missing macro is omitted")
	require.Equal(t, int64(2_820_000), f["current_value_centavos"], "portfolio facts still present")
}

func TestBuildFacts_EmptyPortfolio(t *testing.T) {
	b := NewFactBuilder(
		fakeDashboard{d: dashboard.Dashboard{Allocation: []dashboard.ClassSlice{{Class: dashboard.ClassFII}}}},
		fakeProfile{err: profile.ErrProfileNotFound},
		fakeMacro{},
	)
	f, err := b.BuildFacts(context.Background(), "u1")
	require.NoError(t, err)
	require.False(t, hasHoldings(f), "an all-zero portfolio is empty")
	require.NotContains(t, f, "allocation_bps", "no non-zero classes → omitted")
	require.NotContains(t, f, "fii_sector_bps")
}
