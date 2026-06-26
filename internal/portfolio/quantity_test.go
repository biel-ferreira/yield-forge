package portfolio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQuantity(t *testing.T) {
	t.Run("valid positive", func(t *testing.T) {
		for _, n := range []int{1, 10, 1000} {
			q, err := ParseQuantity(n)
			require.NoError(t, err, "n=%d", n)
			require.Equal(t, n, q.Value())
		}
	})

	t.Run("zero and negative rejected", func(t *testing.T) {
		for _, n := range []int{0, -1, -100} {
			_, err := ParseQuantity(n)
			require.ErrorIs(t, err, ErrInvalidQuantity, "should reject %d", n)
		}
	})
}
