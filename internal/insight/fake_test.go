package insight

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFake(t *testing.T) {
	res, err := Fake{}.Generate(context.Background(), InsightRequest{Facts: Facts{"total_centavos": 1000}})
	require.NoError(t, err)
	require.Len(t, res.Insights, 1)
	require.NotEmpty(t, res.Insights[0].Explanation, "fake output is explainable (passes the gate)")
	require.False(t, containsOrder(insightText(res.Insights[0])), "fake output is order-free")

	_, err = Fake{}.Generate(context.Background(), InsightRequest{Facts: nil})
	require.ErrorIs(t, err, ErrInsufficientFacts)
}
