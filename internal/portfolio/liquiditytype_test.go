package portfolio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLiquidityType(t *testing.T) {
	t.Run("valid (normalized)", func(t *testing.T) {
		cases := map[string]LiquidityType{
			"daily":         LiquidityDaily,
			"At_Maturity":   LiquidityAtMaturity,
			" at_maturity ": LiquidityAtMaturity,
		}
		for in, want := range cases {
			got, err := ParseLiquidityType(in)
			require.NoError(t, err, "input %q", in)
			require.Equal(t, want, got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for _, in := range []string{"", "weekly", "cdb", "liquid"} {
			_, err := ParseLiquidityType(in)
			require.ErrorIs(t, err, ErrInvalidLiquidityType, "should reject %q", in)
		}
	})
}

func TestLiquidityType_RequiresMaturity(t *testing.T) {
	require.True(t, LiquidityAtMaturity.RequiresMaturity())
	require.False(t, LiquidityDaily.RequiresMaturity())
}
