package money

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShareBps(t *testing.T) {
	cases := []struct {
		part, whole int64
		want        int
	}{
		{70, 100, 7000},   // 70%
		{1, 3, 3333},      // 33.33% half-up
		{2, 3, 6667},      // 66.67% half-up
		{1, 2, 5000},      // 50%
		{100, 100, 10000}, // 100%
		{0, 100, 0},       // 0%
		{50, 0, 0},        // divide-by-zero guarded
		{-30, 100, -3000}, // negative (a loss)
	}
	for _, c := range cases {
		require.Equal(t, c.want, ShareBps(c.part, c.whole), "ShareBps(%d, %d)", c.part, c.whole)
	}
}

func TestAccrueSimpleInterest(t *testing.T) {
	cases := []struct {
		principal int64
		rateBps   int
		days      int
		want      int64
	}{
		{1_000_000, 1200, 365, 120_000}, // 12%/yr on R$10k for 1 year = R$1.200,00
		{1_000_000, 1200, 730, 240_000}, // 2 years
		{1_825_000, 1, 1, 1},            // exactly 0.5 centavo -> half-up to 1
		{1_824_999, 1, 1, 0},            // just under 0.5 -> 0
		{1_000_000, 0, 365, 0},          // zero rate
		{1_000_000, 1200, 0, 0},         // zero days
		{0, 1200, 365, 0},               // zero principal
	}
	for _, c := range cases {
		require.Equal(t, c.want, AccrueSimpleInterest(c.principal, c.rateBps, c.days),
			"AccrueSimpleInterest(%d, %d, %d)", c.principal, c.rateBps, c.days)
	}
}

// TestAccrueSimpleInterest_NoOverflow proves a huge principal × rate × days does not overflow
// (the big.Int intermediate), where naive int64 multiplication would wrap.
func TestAccrueSimpleInterest_NoOverflow(t *testing.T) {
	got := AccrueSimpleInterest(1_000_000_000_00, 10_000, 36_500) // R$1B, 100%/yr, 100 years
	require.Positive(t, got)
	require.Equal(t, int64(1_000_000_000_00)*100, got, "100%/yr simple interest over 100y = 100× principal")
}
