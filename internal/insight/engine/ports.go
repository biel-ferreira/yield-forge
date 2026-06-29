package engine

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// The Fact Builder reads its grounding through small consumer interfaces (accept interfaces),
// satisfied at the wiring edge by the dashboard service, the profile service, and the macro
// repository (SPEC-104 FR-1035). The engine reasons only over what these return — computed
// facts, never invented numbers (BR-1041).

// DashboardReader supplies the computed portfolio figures (current value, allocation, sector
// exposure, stale tickers) — SPEC-103.
type DashboardReader interface {
	GetDashboard(ctx context.Context, userID string) (dashboard.Dashboard, error)
}

// ProfileReader supplies the investor profile (risk, objectives, horizon) — SPEC-101. A
// profile.ErrProfileNotFound is handled gracefully (profile facts omitted), not an error.
type ProfileReader interface {
	GetProfile(ctx context.Context, userID string) (profile.Profile, error)
}

// MacroReader supplies the latest macro indicator (SELIC/CDI/IPCA) — SPEC-006. A missing
// indicator is handled gracefully (that fact omitted).
type MacroReader interface {
	GetLatestMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error)
}
