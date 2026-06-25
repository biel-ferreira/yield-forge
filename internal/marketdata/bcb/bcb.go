// Package bcb is the macro-indicator adapter over the Banco Central do Brasil SGS API — a
// free, public, no-key JSON service (SPEC-006 FR-603 / FR-007). It maps SELIC, CDI, and
// IPCA to their SGS series codes and reads the latest observation. HTTP lives here only
// (BR-601); percent values become integer basis points via the money helper (BR-604).
//
// IFIX is NOT a BCB series (it is a B3 index), so this adapter returns ErrProviderUnavailable
// for it; the worker degrades gracefully and a free IFIX source is a documented follow-up
// (SPEC-006 §15).
package bcb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
)

const (
	maxBytes   = 1 << 20 // 1 MiB — the latest-observation payload is tiny
	userAgent  = "YieldForge/1.0 (+https://github.com/biel-ferreira/yield-forge; market-data ingestion)"
	dateLayout = "02/01/2006" // BCB returns dd/mm/yyyy
)

// seriesCodes maps each supported indicator to its SGS series code (all annualized percents):
//   - 432   Meta Selic definida pelo Copom (% a.a.)
//   - 4389  CDI anualizada base 252 (% a.a.)
//   - 13522 IPCA acumulado nos últimos 12 meses (%)
var seriesCodes = map[marketdata.Indicator]int{
	marketdata.IndicatorSELIC: 432,
	marketdata.IndicatorCDI:   4389,
	marketdata.IndicatorIPCA:  13522,
}

// Adapter fetches macro indicators from a BCB-SGS-compatible endpoint.
type Adapter struct {
	baseURL string
	client  *http.Client
	clock   clock.Clock
}

// New returns a BCB adapter. timeout bounds each request (SPEC-006 FR-608); clk stamps FetchedAt.
func New(baseURL string, timeout time.Duration, clk clock.Clock) *Adapter {
	return &Adapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		clock:   clk,
	}
}

// sgsObservation is one SGS data point: {"data":"01/06/2026","valor":"10.50"}.
type sgsObservation struct {
	Data  string `json:"data"`
	Valor string `json:"valor"`
}

// FetchMacroIndicator reads the latest SGS observation for ind. An unsupported indicator
// (IFIX), a transport/parse failure, or an empty series each return a wrapped
// ErrProviderUnavailable so the worker keeps last-known-good (BR-602, FR-610).
func (a *Adapter) FetchMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error) {
	code, ok := seriesCodes[ind]
	if !ok {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb has no series for %s", marketdata.ErrProviderUnavailable, ind)
	}

	url := fmt.Sprintf("%s/bcdata.sgs.%d/dados/ultimos/1?formato=json", a.baseURL, code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb new request: %v", marketdata.ErrProviderUnavailable, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb request: %v", marketdata.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb status %d", marketdata.ErrProviderUnavailable, resp.StatusCode)
	}

	var obs []sgsObservation
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBytes)).Decode(&obs); err != nil {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb decode: %v", marketdata.ErrProviderUnavailable, err)
	}
	if len(obs) == 0 {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb empty series %d", marketdata.ErrProviderUnavailable, code)
	}
	latest := obs[len(obs)-1]

	bps, err := money.DecimalToMinor(latest.Valor, 2) // percent -> basis points
	if err != nil {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb value %q: %v", marketdata.ErrProviderUnavailable, latest.Valor, err)
	}
	refDate, err := time.Parse(dateLayout, latest.Data)
	if err != nil {
		return marketdata.MacroIndicator{}, fmt.Errorf("%w: bcb date %q: %v", marketdata.ErrProviderUnavailable, latest.Data, err)
	}

	return marketdata.MacroIndicator{
		Indicator:     ind,
		Value:         bps,
		Unit:          marketdata.UnitBps,
		ReferenceDate: refDate.UTC(),
		Source:        "bcb-sgs",
		FetchedAt:     a.clock.Now().UTC(),
	}, nil
}
