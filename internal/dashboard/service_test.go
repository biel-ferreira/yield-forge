package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

type fakeHoldings struct {
	h   portfolio.Holdings
	err error
}

func (f fakeHoldings) ListHoldings(context.Context, string) (portfolio.Holdings, error) {
	return f.h, f.err
}

type fakeQuotes struct {
	byTicker map[string]marketdata.FIIQuote
	err      error // a hard (non-not-found) error, when set
}

func (f fakeQuotes) GetFIIQuoteByTicker(_ context.Context, t marketdata.Ticker) (marketdata.FIIQuote, error) {
	if f.err != nil {
		return marketdata.FIIQuote{}, f.err
	}
	q, ok := f.byTicker[t.String()]
	if !ok {
		return marketdata.FIIQuote{}, marketdata.ErrFIIQuoteNotFound
	}
	return q, nil
}

type svcClock struct{ t time.Time }

func (c svcClock) Now() time.Time { return c.t }

func TestService_GetDashboard_FetchesQuotesAndMarksStale(t *testing.T) {
	holdings := portfolio.Holdings{FII: []portfolio.FIIHolding{
		mustFII(t, "HGLG11", 100, 15_750),
		mustFII(t, "XPLG11", 10, 10_000), // no quote → stale
	}}
	svc := NewService(
		fakeHoldings{h: holdings},
		fakeQuotes{byTicker: map[string]marketdata.FIIQuote{
			"HGLG11": quote("HGLG11", 16_000, marketdata.SectorLogistics, 110),
		}},
		svcClock{t: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)},
	)

	d, err := svc.GetDashboard(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, int64(1_700_000), d.Summary.CurrentValueCentavos) // 1.6M(quoted) + 0.1M(stale cost basis)
	require.Equal(t, []string{"XPLG11"}, d.StaleTickers)
}

func TestService_GetDashboard_HoldingsErrorPropagates(t *testing.T) {
	boom := errors.New("db down")
	svc := NewService(fakeHoldings{err: boom}, fakeQuotes{}, svcClock{})
	_, err := svc.GetDashboard(context.Background(), "u1")
	require.ErrorIs(t, err, boom)
}

func TestService_GetDashboard_HardQuoteErrorPropagates(t *testing.T) {
	boom := errors.New("quote store error")
	svc := NewService(
		fakeHoldings{h: portfolio.Holdings{FII: []portfolio.FIIHolding{mustFII(t, "HGLG11", 1, 100)}}},
		fakeQuotes{err: boom}, // a non-not-found error must surface, not be swallowed as stale
		svcClock{t: time.Now()},
	)
	_, err := svc.GetDashboard(context.Background(), "u1")
	require.ErrorIs(t, err, boom)
}

func TestService_GetDashboard_Empty(t *testing.T) {
	svc := NewService(fakeHoldings{}, fakeQuotes{}, svcClock{t: time.Now()})
	d, err := svc.GetDashboard(context.Background(), "u1")
	require.NoError(t, err)
	require.Zero(t, d.Summary.CurrentValueCentavos)
	require.Len(t, d.Allocation, 4)
}
