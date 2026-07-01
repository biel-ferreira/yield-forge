package projection

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

// The service reads its inputs through small consumer interfaces (accept interfaces), satisfied at
// the wiring edge by the dashboard and portfolio services (SPEC-107 §7).

// DashboardReader supplies the computed current value + FII monthly income (SPEC-103).
type DashboardReader interface {
	GetDashboard(ctx context.Context, userID string) (dashboard.Dashboard, error)
}

// HoldingsReader supplies the holdings (SPEC-102) for the fixed-income annual income.
type HoldingsReader interface {
	ListHoldings(ctx context.Context, userID string) (portfolio.Holdings, error)
}
