package profile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHorizon(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, years := range []int{MinHorizonYears, 5, 10, 20, MaxHorizonYears} {
			h, err := ParseHorizon(years)
			require.NoError(t, err, "years %d", years)
			require.Equal(t, years, h.Years())
		}
	})

	t.Run("out of range", func(t *testing.T) {
		for _, years := range []int{0, -1, MaxHorizonYears + 1, 100} {
			_, err := ParseHorizon(years)
			require.ErrorIs(t, err, ErrInvalidHorizon, "should reject %d", years)
		}
	})
}
