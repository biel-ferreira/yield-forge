package ingest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeTickerReader is a hand-written fake tickerReader (no mocking lib, project convention).
type fakeTickerReader struct {
	tickers []string
	err     error
}

func (f fakeTickerReader) DistinctFIITickers(context.Context) ([]string, error) {
	return f.tickers, f.err
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestHoldingsSource_Tickers(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
		want []string // ticker strings, order-sensitive (passthrough order)
	}{
		{
			name: "distinct passthrough parses into Tickers",
			raw:  []string{"HGLG11", "MXRF11"},
			want: []string{"HGLG11", "MXRF11"},
		},
		{
			name: "empty result is a valid empty slice",
			raw:  nil,
			want: []string{},
		},
		{
			name: "malformed entry is skipped, valid ones still returned (BR-075)",
			raw:  []string{"HGLG11", "not-a-ticker", "MXRF11"},
			want: []string{"HGLG11", "MXRF11"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := newHoldingsSource(fakeTickerReader{tickers: tt.raw}, testLogger())
			got, err := src.Tickers(context.Background())
			require.NoError(t, err)
			gotStr := make([]string, len(got))
			for i, tk := range got {
				gotStr[i] = tk.String()
			}
			require.Equal(t, tt.want, gotStr)
		})
	}
}

func TestHoldingsSource_ReaderErrorSurfaced(t *testing.T) {
	boom := errors.New("boom")
	src := newHoldingsSource(fakeTickerReader{err: boom}, testLogger())

	_, err := src.Tickers(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, boom)
}
