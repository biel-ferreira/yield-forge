package portfolio

import (
	"testing"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/stretchr/testify/require"
)

func TestFixedIncomeHolding_EffectiveAnnualRateBps(t *testing.T) {
	macroWithCDIAndIPCA := map[marketdata.Indicator]marketdata.MacroIndicator{
		marketdata.IndicatorCDI:  {Indicator: marketdata.IndicatorCDI, Value: 1050}, // 10.50% a.a.
		marketdata.IndicatorIPCA: {Indicator: marketdata.IndicatorIPCA, Value: 450}, // 4.50% a.a.
	}

	t.Run("Prefixado passes through unchanged, regardless of macro (BR-1093)", func(t *testing.T) {
		h := FixedIncomeHolding{IndexerType: IndexerPrefixado, AnnualRateBps: 1200}
		require.Equal(t, 1200, h.EffectiveAnnualRateBps(macroWithCDIAndIPCA))
		require.Equal(t, 1200, h.EffectiveAnnualRateBps(nil))
	})

	t.Run("CDIPercentual resolves rate x CDI / 10000, half-up", func(t *testing.T) {
		// 120% do CDI, CDI = 10.50% a.a. -> 1050 * 12000 / 10000 = 1260 (12.60% a.a.)
		h := FixedIncomeHolding{IndexerType: IndexerCDIPercentual, AnnualRateBps: 12000}
		require.Equal(t, 1260, h.EffectiveAnnualRateBps(macroWithCDIAndIPCA))
	})

	t.Run("IPCASpread resolves spread + IPCA", func(t *testing.T) {
		// IPCA + 5.80%, IPCA = 4.50% a.a. -> 580 + 450 = 1030 (10.30% a.a.)
		h := FixedIncomeHolding{IndexerType: IndexerIPCASpread, AnnualRateBps: 580}
		require.Equal(t, 1030, h.EffectiveAnnualRateBps(macroWithCDIAndIPCA))
	})

	t.Run("degrades to the raw stored value when the reference indicator is absent (D3)", func(t *testing.T) {
		empty := map[marketdata.Indicator]marketdata.MacroIndicator{}
		cdi := FixedIncomeHolding{IndexerType: IndexerCDIPercentual, AnnualRateBps: 12000}
		require.Equal(t, 12000, cdi.EffectiveAnnualRateBps(empty))
		require.Equal(t, 12000, cdi.EffectiveAnnualRateBps(nil))

		ipca := FixedIncomeHolding{IndexerType: IndexerIPCASpread, AnnualRateBps: 580}
		require.Equal(t, 580, ipca.EffectiveAnnualRateBps(empty))
	})

	t.Run("zero-value Indexer (unset) behaves like Prefixado", func(t *testing.T) {
		h := FixedIncomeHolding{AnnualRateBps: 900}
		require.Equal(t, 900, h.EffectiveAnnualRateBps(macroWithCDIAndIPCA))
	})
}
