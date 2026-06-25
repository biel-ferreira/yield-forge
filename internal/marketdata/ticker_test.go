package marketdata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTicker(t *testing.T) {
	t.Run("valid tickers normalize", func(t *testing.T) {
		cases := map[string]string{
			"HGLG11":    "HGLG11",
			"hglg11":    "HGLG11", // lowercased
			"  knri11 ": "KNRI11", // trimmed
			"MXRF11":    "MXRF11",
		}
		for in, want := range cases {
			got, err := ParseTicker(in)
			require.NoError(t, err, "input %q", in)
			require.Equal(t, want, got.String())
		}
	})

	t.Run("invalid tickers are rejected", func(t *testing.T) {
		bad := []string{"", "HGLG", "HG11", "HGLG111", "HGL1", "12HGLG", "HGLG11.SA", "HG-LG11"}
		for _, in := range bad {
			_, err := ParseTicker(in)
			require.ErrorIs(t, err, ErrInvalidTicker, "should reject %q", in)
		}
	})
}
