// Package yahoo is the FII last-dividend adapter: it reads the most recent cash
// distribution for a B3 ticker from Yahoo Finance's chart API (the `.SA` suffix denotes
// the São Paulo exchange) — SPEC-006 FR-602 / D4. It is a free, no-key, but UNOFFICIAL
// endpoint, so it is treated as best-effort: a failure yields ErrProviderUnavailable and
// the composite simply omits last-dividend rather than failing the whole quote. HTTP lives
// here only (BR-601); amounts become centavos via the money helper (BR-604).
package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
)

const (
	maxBytes  = 2 << 20 // 2 MiB — the chart payload is small
	userAgent = "YieldForge/1.0 (+https://github.com/biel-ferreira/yield-forge; market-data ingestion)"
)

// Adapter reads last-dividend data from a Yahoo-compatible chart endpoint.
type Adapter struct {
	baseURL string
	client  *http.Client
}

// New returns a Yahoo adapter. timeout bounds each request (SPEC-006 FR-608).
func New(baseURL string, timeout time.Duration) *Adapter {
	return &Adapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

// chartResponse is the slice of Yahoo's chart payload we need.
type chartResponse struct {
	Chart struct {
		Result []struct {
			Events struct {
				Dividends map[string]struct {
					Amount json.Number `json:"amount"`
					Date   int64       `json:"date"`
				} `json:"dividends"`
			} `json:"events"`
		} `json:"result"`
	} `json:"chart"`
}

// FetchLastDividend returns the most recent distribution (centavos) and its date for t.
// When the source has no dividends it returns (0, nil, nil); on any transport/parse failure
// it returns a wrapped ErrProviderUnavailable (the caller treats last-dividend as optional).
func (a *Adapter) FetchLastDividend(ctx context.Context, t marketdata.Ticker) (int64, *time.Time, error) {
	url := fmt.Sprintf("%s/v8/finance/chart/%s.SA?range=1y&interval=1d&events=div", a.baseURL, t.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: yahoo new request: %v", marketdata.ErrProviderUnavailable, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: yahoo request: %v", marketdata.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("%w: yahoo status %d", marketdata.ErrProviderUnavailable, resp.StatusCode)
	}

	var cr chartResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBytes)).Decode(&cr); err != nil {
		return 0, nil, fmt.Errorf("%w: yahoo decode: %v", marketdata.ErrProviderUnavailable, err)
	}
	if len(cr.Chart.Result) == 0 {
		return 0, nil, nil // no data for this ticker — not an error
	}

	dividends := cr.Chart.Result[0].Events.Dividends
	if len(dividends) == 0 {
		return 0, nil, nil
	}

	// Pick the most recent distribution by date.
	var latestDate int64
	var latestAmount json.Number
	for _, d := range dividends {
		if d.Date >= latestDate {
			latestDate, latestAmount = d.Date, d.Amount
		}
	}

	cents, err := money.DecimalToMinor(latestAmount.String(), 2)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: yahoo amount %q: %v", marketdata.ErrProviderUnavailable, latestAmount.String(), err)
	}
	day := time.Unix(latestDate, 0).UTC()
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	return cents, &day, nil
}
