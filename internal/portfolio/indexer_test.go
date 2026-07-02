package portfolio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIndexer(t *testing.T) {
	t.Run("valid (normalized)", func(t *testing.T) {
		cases := map[string]Indexer{
			"prefixado":      IndexerPrefixado,
			"CDI_Percentual": IndexerCDIPercentual,
			" ipca_spread ":  IndexerIPCASpread,
		}
		for in, want := range cases {
			got, err := ParseIndexer(in)
			require.NoError(t, err, "input %q", in)
			require.Equal(t, want, got)
		}
	})

	t.Run("empty defaults to Prefixado (BR-1093 backward compatibility)", func(t *testing.T) {
		got, err := ParseIndexer("")
		require.NoError(t, err)
		require.Equal(t, IndexerPrefixado, got)
	})

	t.Run("invalid", func(t *testing.T) {
		for _, in := range []string{"cdi", "ipca", "fixed", "% do cdi"} {
			_, err := ParseIndexer(in)
			require.ErrorIs(t, err, ErrInvalidIndexer, "should reject %q", in)
		}
	})
}
