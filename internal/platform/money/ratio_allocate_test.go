package money

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllocateBps_SumsToExactly10000(t *testing.T) {
	tests := []struct {
		name    string
		weights []int64
		want    []int
	}{
		{"even split of three (largest-remainder)", []int64{1, 1, 1}, []int{3334, 3333, 3333}},
		{"seventy-thirty", []int64{7000, 3000}, []int{7000, 3000}},
		{"single weight gets all", []int64{42}, []int{10000}},
		{"zero weights → all zero", []int64{0, 0}, []int{0, 0}},
		{"negative treated as zero", []int64{-5, 10}, []int{0, 10000}},
		{"empty", []int64{}, []int{}},
		{"six equal weights, leftover spread by index", []int64{1, 1, 1, 1, 1, 1}, []int{1667, 1667, 1667, 1667, 1666, 1666}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AllocateBps(tc.weights)
			require.Equal(t, tc.want, got)

			sum := 0
			for _, s := range got {
				sum += s
			}
			if len(tc.weights) > 0 && hasPositive(tc.weights) {
				require.Equal(t, 10000, sum, "shares must reconcile to exactly 10000")
			} else {
				require.Equal(t, 0, sum)
			}
		})
	}
}

func TestAllocateBps_Deterministic(t *testing.T) {
	w := []int64{1700000, 1120000, 0}
	require.Equal(t, AllocateBps(w), AllocateBps(w), "same weights → same split")
}

func TestApplyBps(t *testing.T) {
	require.Equal(t, int64(5000), ApplyBps(10000, 5000), "50% of R$100,00")
	require.Equal(t, int64(33), ApplyBps(100, 3333), "33.33% rounds half-up to 33")
	require.Equal(t, int64(2), ApplyBps(3, 5000), "1.5 rounds half-up to 2")
	require.Equal(t, int64(0), ApplyBps(0, 5000))
	require.Equal(t, int64(0), ApplyBps(1000, 0))
}

func hasPositive(ws []int64) bool {
	for _, w := range ws {
		if w > 0 {
			return true
		}
	}
	return false
}
