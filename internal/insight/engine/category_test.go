package engine

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

func TestCategory_Task(t *testing.T) {
	require.Equal(t, insight.Task("portfolio"), CategoryPortfolio.Task())
	require.Equal(t, insight.Task("allocation"), CategoryAllocation.Task())
	require.Equal(t, insight.Task("market_context"), CategoryMarketContext.Task())
	require.Len(t, AllCategories, 3)
}
