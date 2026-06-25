// Command ingest performs a single market-data ingestion pass and exits — the
// request-decoupled, cron-friendly alternative to the in-process scheduler (SPEC-006
// FR-604, D2). It exits non-zero only on a fatal config/DB error, never on a per-item
// provider failure (those degrade and keep last-known-good).
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/biel-ferreira/yield-forge/internal/marketdata/ingest"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/platform/logging"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).
			Error("configuration error", slog.String("error", err.Error()))
		return err
	}

	logger := logging.New(cfg)
	for _, warning := range cfg.Warnings {
		logger.Warn("configuration warning", slog.String("detail", warning))
	}

	ctx := context.Background()

	// Telemetry (no-op unless configured); flush LAST so ingestion spans/metrics export.
	shutdownTelemetry, err := observability.Setup(ctx, cfg, buildinfo.Version)
	if err != nil {
		logger.Error("telemetry setup failed", slog.String("error", err.Error()))
		return err
	}
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if serr := shutdownTelemetry(sctx); serr != nil {
			logger.Error("telemetry shutdown", slog.String("error", serr.Error()))
		}
	}()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		logger.Error("database connection failed", slog.String("error", err.Error()))
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("closing database pool", slog.String("error", cerr.Error()))
		}
	}()

	worker, err := ingest.New(cfg, db, logger, clock.System{})
	if err != nil {
		logger.Error("market data ingestion setup failed", slog.String("error", err.Error()))
		return err
	}

	logger.Info("market data ingestion starting",
		slog.String("provider", cfg.MarketDataProvider),
		slog.Int("watchlist_size", len(cfg.MarketDataWatchlist)),
	)
	worker.RunOnce(ctx)
	return nil
}
