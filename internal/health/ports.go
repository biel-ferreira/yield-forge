package health

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// The service reads its inputs through small consumer interfaces (accept interfaces), satisfied at
// the wiring edge by the dashboard, profile, portfolio, and market-data services (SPEC-106 §7).

// DashboardReader supplies the computed allocation / sector exposure / concentration (SPEC-103).
type DashboardReader interface {
	GetDashboard(ctx context.Context, userID string) (dashboard.Dashboard, error)
}

// ProfileReader supplies the investor profile (SPEC-101); a not-set profile degrades gracefully.
type ProfileReader interface {
	GetProfile(ctx context.Context, userID string) (profile.Profile, error)
}

// HoldingsReader supplies the holdings (SPEC-102) for the position count and the fixed-income
// liquidity split.
type HoldingsReader interface {
	ListHoldings(ctx context.Context, userID string) (portfolio.Holdings, error)
}

// MacroReader supplies the latest macro indicator (SPEC-006) for the market-aware factors; a
// missing indicator degrades to a neutral tilt.
type MacroReader interface {
	GetLatestMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error)
}
