// Command api is the entrypoint for the YieldForge HTTP API.
//
// It loads configuration, builds the logger, surfaces any config warnings, wires
// the HTTP router, and runs the server until interrupted (graceful shutdown).
package main

import (
	"log/slog"
	"os"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/httpserver"
	"github.com/biel-ferreira/yield-forge/internal/platform/logging"
	transporthttp "github.com/biel-ferreira/yield-forge/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// The logger isn't built yet, so report the config failure on stderr.
		bootstrap := slog.New(slog.NewTextHandler(os.Stderr, nil))
		bootstrap.Error("configuration error", slog.String("error", err.Error()))
		os.Exit(1)
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

	router := transporthttp.NewRouter(logger, buildinfo.Get())

	if err := httpserver.Run(cfg, router, logger); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
