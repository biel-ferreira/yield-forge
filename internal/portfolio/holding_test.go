package portfolio

import (
	"testing"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/stretchr/testify/require"
)

func TestFixedIncomeHolding_ResolveEffectiveRate(t *testing.T) {
	macroWithCDIAndIPCA := map[marketdata.Indicator]marketdata.MacroIndicator{
		marketdata.IndicatorCDI:  {Indicator: marketdata.IndicatorCDI, Value: 1050}, // 10.50% a.a.
		marketdata.IndicatorIPCA: {Indicator: marketdata.IndicatorIPCA, Value: 450}, // 4.50% a.a.
	}

	t.Run("Prefixado passes through unchanged, regardless of macro (BR-1093)", func(t *testing.T) {
		h := FixedIncomeHolding{IndexerType: IndexerPrefixado, AnnualRateBps: 1200}
		require.Equal(t, 1200, h.ResolveEffectiveRate(macroWithCDIAndIPCA))
		require.Equal(t, 1200, h.ResolveEffectiveRate(nil))
	})

	t.Run("CDIPercentual resolves rate x CDI / 10000, half-up", func(t *testing.T) {
		// 120% do CDI, CDI = 10.50% a.a. -> 1050 * 12000 / 10000 = 1260 (12.60% a.a.)
		h := FixedIncomeHolding{IndexerType: IndexerCDIPercentual, AnnualRateBps: 12000}
		require.Equal(t, 1260, h.ResolveEffectiveRate(macroWithCDIAndIPCA))
	})

	t.Run("IPCASpread resolves spread + IPCA", func(t *testing.T) {
		// IPCA + 5.80%, IPCA = 4.50% a.a. -> 580 + 450 = 1030 (10.30% a.a.)
		h := FixedIncomeHolding{IndexerType: IndexerIPCASpread, AnnualRateBps: 580}
		require.Equal(t, 1030, h.ResolveEffectiveRate(macroWithCDIAndIPCA))
	})

	t.Run("degrades to the raw stored value when the reference indicator is absent (D3)", func(t *testing.T) {
		empty := map[marketdata.Indicator]marketdata.MacroIndicator{}
		cdi := FixedIncomeHolding{IndexerType: IndexerCDIPercentual, AnnualRateBps: 12000}
		require.Equal(t, 12000, cdi.ResolveEffectiveRate(empty))
		require.Equal(t, 12000, cdi.ResolveEffectiveRate(nil))

		ipca := FixedIncomeHolding{IndexerType: IndexerIPCASpread, AnnualRateBps: 580}
		require.Equal(t, 580, ipca.ResolveEffectiveRate(empty))
	})

	t.Run("zero-value Indexer (unset) behaves like Prefixado", func(t *testing.T) {
		h := FixedIncomeHolding{AnnualRateBps: 900}
		require.Equal(t, 900, h.ResolveEffectiveRate(macroWithCDIAndIPCA))
	})
}

func TestFixedIncomeHolding_EstimateInterest(t *testing.T) {
	t.Run("zero elapsed days accrues nothing", func(t *testing.T) {
		now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
		h := FixedIncomeHolding{InvestedAmountCentavos: 1_000_000, EffectiveAnnualRateBps: 1200, LastReconciledAt: now}
		require.Equal(t, int64(0), h.EstimateInterest(now))
	})

	t.Run("positive elapsed days accrues via the shared AccrueSimpleInterest formula", func(t *testing.T) {
		last := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
		now := last.AddDate(0, 0, 365) // exactly one year
		h := FixedIncomeHolding{InvestedAmountCentavos: 1_000_000, EffectiveAnnualRateBps: 1200, LastReconciledAt: last}
		require.Equal(t, int64(120_000), h.EstimateInterest(now)) // 12%/yr on R$10,000 = R$1,200
	})
}

func TestFixedIncomeHolding_IsReconciliationDue(t *testing.T) {
	t.Run("reconciled this calendar month is not due", func(t *testing.T) {
		h := FixedIncomeHolding{LastReconciledAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
		now := time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC)
		require.False(t, h.IsReconciliationDue(now))
	})

	t.Run("reconciled today is not due", func(t *testing.T) {
		now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
		h := FixedIncomeHolding{LastReconciledAt: now}
		require.False(t, h.IsReconciliationDue(now))
	})

	t.Run("last reconciled in the prior month, even one day into the new month, is due", func(t *testing.T) {
		h := FixedIncomeHolding{LastReconciledAt: time.Date(2026, 6, 30, 23, 59, 0, 0, time.UTC)}
		now := time.Date(2026, 7, 1, 0, 1, 0, 0, time.UTC)
		require.True(t, h.IsReconciliationDue(now))
	})

	t.Run("last reconciled in a prior year is due", func(t *testing.T) {
		h := FixedIncomeHolding{LastReconciledAt: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)}
		now := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
		require.True(t, h.IsReconciliationDue(now))
	})
}
