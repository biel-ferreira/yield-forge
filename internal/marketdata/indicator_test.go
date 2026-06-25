package marketdata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIndicator(t *testing.T) {
	t.Run("valid (case-insensitive)", func(t *testing.T) {
		for _, in := range []string{"selic", "SELIC", "Ipca", "cdi", "IFIX"} {
			_, err := ParseIndicator(in)
			require.NoError(t, err, "input %q", in)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for _, in := range []string{"", "dollar", "igpm", "selicc"} {
			_, err := ParseIndicator(in)
			require.ErrorIs(t, err, ErrInvalidIndicator, "should reject %q", in)
		}
	})
}

func TestIndicatorDefaultUnit(t *testing.T) {
	require.Equal(t, UnitBps, IndicatorSELIC.DefaultUnit())
	require.Equal(t, UnitBps, IndicatorIPCA.DefaultUnit())
	require.Equal(t, UnitBps, IndicatorCDI.DefaultUnit())
	require.Equal(t, UnitPoints, IndicatorIFIX.DefaultUnit(), "IFIX is an index level, not a rate")
}
