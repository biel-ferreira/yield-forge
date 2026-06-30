package rebalancing

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

func mustContribution(t *testing.T, centavos int64) Contribution {
	t.Helper()
	c, err := ParseContribution(centavos)
	require.NoError(t, err)
	return c
}

func sumShares(areas []AreaShare) int {
	sum := 0
	for _, a := range areas {
		sum += a.SuggestedShareBps
	}
	return sum
}

func TestParseContribution(t *testing.T) {
	_, err := ParseContribution(0)
	require.ErrorIs(t, err, ErrInvalidContribution)
	_, err = ParseContribution(-100)
	require.ErrorIs(t, err, ErrInvalidContribution)

	c, err := ParseContribution(50_000)
	require.NoError(t, err)
	require.Equal(t, int64(50_000), c.Centavos())
}

func TestSplit_SteersTowardUnderweightClass(t *testing.T) {
	// Moderate target is 50/50. The portfolio is 100% FII, so the contribution should go to
	// Fixed Income (the under-weight class) — steering toward balance.
	alloc := Allocation{
		TotalCentavos: 100_000,
		ByClass:       map[dashboard.AssetClass]int64{dashboard.ClassFII: 100_000},
	}
	areas := Split(alloc, profile.RiskModerate, mustContribution(t, 100_000))

	require.Equal(t, 10000, sumShares(areas), "shares reconcile to exactly 10000")
	require.Len(t, areas, 1, "only the under-weight class is funded")
	require.Equal(t, dashboard.ClassFixedIncome, areas[0].Class)
	require.Equal(t, 10000, areas[0].SuggestedShareBps)
	require.Equal(t, int64(100_000), areas[0].SuggestedAmountCentavos)
}

func TestSplit_ConservativeTiltsFixedIncome(t *testing.T) {
	// Empty portfolio + conservative: the split follows the profile-implied direction (70% FI).
	areas := Split(Allocation{}, profile.RiskConservative, mustContribution(t, 1_000_000))

	require.Equal(t, 10000, sumShares(areas))
	byClass := map[dashboard.AssetClass]int{}
	for _, a := range areas {
		byClass[a.Class] = a.SuggestedShareBps
	}
	require.Equal(t, 7000, byClass[dashboard.ClassFixedIncome], "conservative tilts Fixed Income")
	require.Equal(t, 3000, byClass[dashboard.ClassFII])
}

func TestSplit_Deterministic(t *testing.T) {
	alloc := Allocation{
		TotalCentavos: 2_820_000,
		ByClass:       map[dashboard.AssetClass]int64{dashboard.ClassFII: 1_700_000, dashboard.ClassFixedIncome: 1_120_000},
	}
	c := mustContribution(t, 50_000)
	require.Equal(t, Split(alloc, profile.RiskModerate, c), Split(alloc, profile.RiskModerate, c))
}

func TestSplit_BalancedPortfolioSplitsTowardTarget(t *testing.T) {
	// Already at the moderate 50/50 mix: new money keeps the balance, so the split is ~50/50
	// (proportional gaps), reconciling to 10000.
	alloc := Allocation{
		TotalCentavos: 100_000,
		ByClass:       map[dashboard.AssetClass]int64{dashboard.ClassFII: 50_000, dashboard.ClassFixedIncome: 50_000},
	}
	areas := Split(alloc, profile.RiskModerate, mustContribution(t, 100_000))
	require.Equal(t, 10000, sumShares(areas))
	byClass := map[dashboard.AssetClass]int{}
	for _, a := range areas {
		byClass[a.Class] = a.SuggestedShareBps
	}
	require.Equal(t, 5000, byClass[dashboard.ClassFII])
	require.Equal(t, 5000, byClass[dashboard.ClassFixedIncome])
}
