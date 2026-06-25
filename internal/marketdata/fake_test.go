package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFake_FIIQuotes(t *testing.T) {
	at := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	hglg := MustParseTicker("HGLG11")

	got, err := Fake{At: at}.FetchFIIQuotes(context.Background(), []Ticker{hglg})
	require.NoError(t, err)
	require.Len(t, got, 1)
	q := got[hglg]
	require.Equal(t, hglg, q.Ticker)
	require.Positive(t, q.PriceCentavos)
	require.Equal(t, "fake", q.Source)
	require.Equal(t, at, q.FetchedAt)
	require.NotNil(t, q.LastDividendDate)
}

func TestFake_MacroIndicator(t *testing.T) {
	selic, err := Fake{}.FetchMacroIndicator(context.Background(), IndicatorSELIC)
	require.NoError(t, err)
	require.Equal(t, UnitBps, selic.Unit)
	require.Positive(t, selic.Value)

	ifix, err := Fake{}.FetchMacroIndicator(context.Background(), IndicatorIFIX)
	require.NoError(t, err)
	require.Equal(t, UnitPoints, ifix.Unit, "IFIX is an index level, not a rate")
}
