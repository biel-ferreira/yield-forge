// Command api is the entrypoint for the YieldForge HTTP API.
//
// It loads configuration, builds the logger, surfaces any config warnings, wires
// the HTTP router, and runs the server until interrupted (graceful shutdown).
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	authbcrypt "github.com/biel-ferreira/yield-forge/internal/auth/bcrypt"
	authpostgres "github.com/biel-ferreira/yield-forge/internal/auth/postgres"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/ingest"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/platform/httpserver"
	"github.com/biel-ferreira/yield-forge/internal/platform/logging"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
	"github.com/biel-ferreira/yield-forge/internal/profile"
	profilepostgres "github.com/biel-ferreira/yield-forge/internal/profile/postgres"
	transporthttp "github.com/biel-ferreira/yield-forge/internal/transport/http"
)

func main() {
	// run does the real work and returns an error; main just maps it to an exit
	// code. This indirection lets deferred cleanup (closing the DB pool) actually
	// run, which os.Exit would otherwise skip.
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		// The logger isn't built yet, so report the config failure on stderr.
		bootstrap := slog.New(slog.NewTextHandler(os.Stderr, nil))
		bootstrap.Error("configuration error", slog.String("error", err.Error()))
		return err
	}

	logger := logging.New(cfg)

	// Surface non-fatal config notes (e.g. an invalid LOG_LEVEL that fell back).
	for _, warning := range cfg.Warnings {
		logger.Warn("configuration warning", slog.String("detail", warning))
	}

	logger.Info("starting yield-forge",
		slog.String("env", cfg.AppEnv),
		slog.String("version", buildinfo.Version),
	)

	ctx := context.Background()

	// Observability (SPEC-004): set up OpenTelemetry. With no exporter configured this
	// is a no-op (the app runs identically). Register its shutdown FIRST so it flushes
	// LAST — after the HTTP server drains and the DB pool closes (defers run LIFO).
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
	logger.Info("telemetry configured",
		slog.String("exporter", cfg.OTELExporterKind),
		slog.Bool("enabled", cfg.TelemetryEnabled()),
	)

	// Connect to PostgreSQL and fail fast if it is unreachable. The pool is closed
	// after the HTTP server drains (the deferred Close runs once run returns).
	logger.Info("connecting to database",
		slog.String("target", cfg.RedactedDatabaseURL()),
		slog.Int("max_open_conns", cfg.DBMaxOpenConns),
		slog.Int("max_idle_conns", cfg.DBMaxIdleConns),
	)
	db, err := database.Connect(ctx, cfg)
	if err != nil {
		logger.Error("database connection failed", slog.String("error", err.Error()))
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("closing database pool", slog.String("error", cerr.Error()))
			return
		}
		logger.Info("database pool closed")
	}()
	logger.Info("database connected")

	// Authentication (SPEC-003): wire the repositories, hasher, and clock into the
	// service, then hand it to the router.
	authService := auth.NewService(
		authpostgres.NewUserRepository(db),
		authpostgres.NewSessionRepository(db),
		authbcrypt.New(),
		clock.System{},
		cfg.SessionTTL,
	)

	// Investor Profile (SPEC-101): the service over its Postgres repository.
	profileService := profile.NewService(profilepostgres.NewProfileRepository(db), clock.System{})

	router := transporthttp.NewRouter(transporthttp.Deps{
		Logger:       logger,
		Build:        buildinfo.Get(),
		Ready:        db,
		Auth:         authService,
		Profile:      profileService,
		CookieName:   cfg.AuthCookieName,
		CookieSecure: cfg.CookieSecure(),
		SessionTTL:   cfg.SessionTTL,
	})

	// Market data ingestion (SPEC-006): build the worker and, when enabled, run the
	// in-process scheduler alongside the server. It is stopped and drained explicitly
	// below — before the deferred DB pool close — so an in-flight run never hits a closed
	// pool. Multi-replica deployments set MARKETDATA_SCHEDULER_ENABLED=false and use
	// cmd/ingest via cron instead, to avoid duplicate ingestion.
	mdWorker, err := ingest.New(cfg, db, logger, clock.System{})
	if err != nil {
		logger.Error("market data ingestion setup failed", slog.String("error", err.Error()))
		return err
	}
	schedulerCtx, stopScheduler := context.WithCancel(ctx)
	schedulerDone := make(chan struct{})
	if cfg.MarketDataSchedulerEnabled {
		scheduler := ingest.NewScheduler(mdWorker, cfg.MarketDataRefreshInterval, logger)
		go func() {
			defer close(schedulerDone)
			scheduler.Run(schedulerCtx)
		}()
		logger.Info("market data scheduler started",
			slog.String("provider", cfg.MarketDataProvider),
			slog.Duration("interval", cfg.MarketDataRefreshInterval))
	} else {
		close(schedulerDone)
		logger.Info("market data scheduler disabled (use cmd/ingest)")
	}

	serveErr := httpserver.Run(ctx, cfg, router, logger)

	// Stop the scheduler and wait for any in-flight run before the deferred DB close runs.
	stopScheduler()
	<-schedulerDone

	if serveErr != nil {
		logger.Error("server error", slog.String("error", serveErr.Error()))
		return serveErr
	}
	return nil
}
