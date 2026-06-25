package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/bcb"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/fii"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/fundamentus"
	mdpostgres "github.com/biel-ferreira/yield-forge/internal/marketdata/postgres"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/yahoo"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

// New builds the ingestion Worker from config: the configured provider (fake | live), the
// Postgres repositories, and the watchlist TickerSource. An invalid watchlist ticker fails
// fast here (SPEC-006 FR-604). Nothing runs until RunOnce/the scheduler is invoked.
func New(cfg config.Config, db *sql.DB, logger *slog.Logger, clk clock.Clock) (*Worker, error) {
	watchlist, err := marketdata.NewWatchlist(cfg.MarketDataWatchlist)
	if err != nil {
		return nil, fmt.Errorf("market data watchlist: %w", err)
	}
	return newWorker(
		buildProvider(cfg, clk),
		mdpostgres.NewFIIQuoteRepository(db),
		mdpostgres.NewMacroRepository(db),
		watchlist,
		clk,
		logger,
	), nil
}

// buildProvider selects the MarketDataProvider from config. "live" composes Fundamentus +
// Yahoo (FII) with BCB-SGS (macro); anything else is the deterministic Fake (the default),
// so the zero-config app and CI never hit the network (SPEC-006 D4, FR-611).
func buildProvider(cfg config.Config, clk clock.Clock) marketdata.MarketDataProvider {
	if cfg.MarketDataProvider == "live" {
		fiiSource := fii.New(
			fundamentus.New(cfg.MarketDataFundamentusBaseURL, cfg.MarketDataTimeout, clk),
			yahoo.New(cfg.MarketDataYahooBaseURL, cfg.MarketDataTimeout),
		)
		macroSource := bcb.New(cfg.MarketDataBCBBaseURL, cfg.MarketDataTimeout, clk)
		return combined{fii: fiiSource, macro: macroSource}
	}
	return marketdata.Fake{At: clk.Now()}
}

// fiiFetcher / macroFetcher are the two halves of the port; combined joins independent FII
// and macro sources into one MarketDataProvider.
type fiiFetcher interface {
	FetchFIIQuotes(ctx context.Context, tickers []marketdata.Ticker) (map[marketdata.Ticker]marketdata.FIIQuote, error)
}

type macroFetcher interface {
	FetchMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error)
}

type combined struct {
	fii   fiiFetcher
	macro macroFetcher
}

var _ marketdata.MarketDataProvider = combined{}

func (c combined) FetchFIIQuotes(ctx context.Context, tickers []marketdata.Ticker) (map[marketdata.Ticker]marketdata.FIIQuote, error) {
	return c.fii.FetchFIIQuotes(ctx, tickers)
}

func (c combined) FetchMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error) {
	return c.macro.FetchMacroIndicator(ctx, ind)
}
