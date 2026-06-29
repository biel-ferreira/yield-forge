package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// macroFacts is the macro set grounded into every fact snapshot (SELIC/CDI/IPCA; IFIX is the
// documented SPEC-006 gap). A missing indicator is omitted, not an error.
var macroFacts = []marketdata.Indicator{marketdata.IndicatorSELIC, marketdata.IndicatorCDI, marketdata.IndicatorIPCA}

// FactBuilder composes the deterministic fact snapshot for a user from the dashboard, profile,
// and macro seams (SPEC-104 FR-1041). It is the PUBLISHED grounding seam (BuildFacts): the
// Conversational Copilot (SPEC-108) and SPEC-105/106 reuse it. All money is int64 centavos and
// all rates/shares integer basis points — never float (BR-1044); the same inputs always yield
// the same facts (BR-1041). It produces only computed values — the LLM never invents numbers.
type FactBuilder struct {
	dashboards DashboardReader
	profiles   ProfileReader
	macro      MacroReader
}

// NewFactBuilder builds a FactBuilder over the read seams.
func NewFactBuilder(d DashboardReader, p ProfileReader, m MacroReader) *FactBuilder {
	return &FactBuilder{dashboards: d, profiles: p, macro: m}
}

// BuildFacts assembles the deterministic grounding facts for userID. The profile and macro
// reads degrade gracefully (a not-set profile or a missing indicator is omitted, not an
// error); a dashboard error is surfaced.
func (b *FactBuilder) BuildFacts(ctx context.Context, userID string) (insight.Facts, error) {
	d, err := b.dashboards.GetDashboard(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("build facts: dashboard: %w", err)
	}

	facts := insight.Facts{
		"total_invested_centavos": d.Summary.TotalInvestedCentavos,
		"current_value_centavos":  d.Summary.CurrentValueCentavos,
		"monthly_income_centavos": d.Summary.MonthlyIncomeCentavos,
		"growth_bps":              d.Summary.GrowthBps,
	}

	// Allocation by class (only the classes that hold value).
	allocation := map[string]int{}
	for _, s := range d.Allocation {
		if s.ValueCentavos > 0 {
			allocation[string(s.Class)] = s.ShareBps
		}
	}
	if len(allocation) > 0 {
		facts["allocation_bps"] = allocation
	}

	// FII sector exposure + concentration signals.
	sectors := map[string]int{}
	largestSectorBps := 0
	for _, s := range d.FIISectors {
		sectors[string(s.Sector)] = s.ShareBps
		if s.ShareBps > largestSectorBps {
			largestSectorBps = s.ShareBps
		}
	}
	if len(sectors) > 0 {
		facts["fii_sector_bps"] = sectors
		facts["fii_sector_count"] = len(sectors)
		facts["largest_fii_sector_bps"] = largestSectorBps
	}

	if len(d.StaleTickers) > 0 {
		facts["stale_tickers"] = d.StaleTickers
	}

	if err := b.addProfileFacts(ctx, userID, facts); err != nil {
		return nil, err
	}
	if err := b.addMacroFacts(ctx, facts); err != nil {
		return nil, err
	}
	return facts, nil
}

// addProfileFacts adds the investor profile; a not-set profile is omitted, not an error.
func (b *FactBuilder) addProfileFacts(ctx context.Context, userID string, facts insight.Facts) error {
	p, err := b.profiles.GetProfile(ctx, userID)
	if errors.Is(err, profile.ErrProfileNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("build facts: profile: %w", err)
	}
	objectives := make([]string, len(p.Objectives))
	for i, o := range p.Objectives {
		objectives[i] = string(o)
	}
	facts["risk_profile"] = string(p.Risk)
	facts["objectives"] = objectives
	facts["horizon_years"] = p.Horizon.Years()
	return nil
}

// addMacroFacts adds the latest SELIC/CDI/IPCA (basis points); a missing indicator is omitted.
func (b *FactBuilder) addMacroFacts(ctx context.Context, facts insight.Facts) error {
	macro := map[string]int64{}
	for _, ind := range macroFacts {
		m, err := b.macro.GetLatestMacroIndicator(ctx, ind)
		if errors.Is(err, marketdata.ErrMacroNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("build facts: macro %s: %w", ind, err)
		}
		macro[string(ind)+"_bps"] = m.Value
	}
	if len(macro) > 0 {
		facts["macro_bps"] = macro
	}
	return nil
}

// hasHoldings reports whether the facts describe a non-empty portfolio (something to analyse).
// The engine uses it to short-circuit the empty-portfolio state without an LLM call (FR-1047).
func hasHoldings(facts insight.Facts) bool {
	invested, _ := facts["total_invested_centavos"].(int64)
	current, _ := facts["current_value_centavos"].(int64)
	return invested != 0 || current != 0
}
