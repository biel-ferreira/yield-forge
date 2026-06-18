// Command api is the entrypoint for the YieldForge HTTP API.
//
// It loads configuration, builds the logger, surfaces any config warnings, wires
// the HTTP router, and runs the server until interrupted (graceful shutdown).
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/platform/httpserver"
	"github.com/biel-ferreira/yield-forge/internal/platform/logging"
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

	router := transporthttp.NewRouter(logger, buildinfo.Get())

	if err := httpserver.Run(ctx, cfg, router, logger); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
		return err
	}
	return nil
}
