package money

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWeightedMeanBps(t *testing.T) {
	require.Equal(t, 100, WeightedMeanBps([]int{100, 100}, []int{5000, 5000}))
	require.Equal(t, 50, WeightedMeanBps([]int{0, 100}, []int{5000, 5000}))
	require.Equal(t, 80, WeightedMeanBps([]int{100, 50}, []int{6000, 4000}), "0.6*100 + 0.4*50 = 80")
	require.Equal(t, 0, WeightedMeanBps(nil, nil))
	// half-up: 0.5*55 + 0.5*56 = 55.5 → 56
	require.Equal(t, 56, WeightedMeanBps([]int{55, 56}, []int{5000, 5000}))
	// mismatched lengths: extra weight ignored
	require.Equal(t, 100, WeightedMeanBps([]int{100}, []int{10000, 1234}))
}
