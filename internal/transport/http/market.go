package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// MarketDataReader is the slice of the market-data reader the transport needs (consumer-
// defined); *marketdata/postgres.MacroRepository satisfies it directly — the same port
// portfolio/health/insight-engine already depend on (SPEC-109 FR-1095).
type MarketDataReader interface {
	GetLatestMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error)
}

type marketHandler struct {
	reader MarketDataReader
	logger *slog.Logger
}

type marketIndicatorResponse struct {
	Indicator     string `json:"indicator"`
	ValueBps      int64  `json:"value_bps"`
	ReferenceDate string `json:"reference_date"` // "YYYY-MM-DD"
}

// publishedIndicators are the reference rates SPEC-109 exposes — SELIC, CDI, IPCA. IFIX (also
// ingested, SPEC-006) is not a rate and isn't needed by any indexer, so it's excluded here.
var publishedIndicators = []marketdata.Indicator{
	marketdata.IndicatorSELIC, marketdata.IndicatorCDI, marketdata.IndicatorIPCA,
}

// getMarketIndicators returns the latest SELIC/CDI/IPCA (SPEC-109 FR-1095) — global reference
// data, no user_id (BR-1095). An indicator with no ingested value yet is silently omitted
// (degrades gracefully, mirroring BR-1094) rather than failing the whole response.
func (h marketHandler) getMarketIndicators(w http.ResponseWriter, r *http.Request) {
	if _, ok := callerID(w, r); !ok {
		return
	}
	out := make([]marketIndicatorResponse, 0, len(publishedIndicators))
	for _, ind := range publishedIndicators {
		m, err := h.reader.GetLatestMacroIndicator(r.Context(), ind)
		if err != nil {
			continue // not yet ingested — omit, don't fail the request (BR-1094)
		}
		out = append(out, marketIndicatorResponse{
			Indicator: string(ind), ValueBps: m.Value, ReferenceDate: m.ReferenceDate.Format(dateLayout),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
