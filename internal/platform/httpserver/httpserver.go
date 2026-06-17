package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

// Run starts an HTTP server with the given handler and blocks until ctx is
// cancelled or the process receives an interrupt or SIGTERM, then shuts down
// gracefully within cfg.ShutdownTimeout. It returns the first non-nil error from
// serving or shutdown (nil on a clean stop).
func Run(ctx context.Context, cfg config.Config, handler http.Handler, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Shut down on ctx cancellation, SIGINT (Ctrl-C), or SIGTERM.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", slog.String("addr", cfg.Addr()))
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil // expected on graceful shutdown
		}
		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		// Server stopped before any signal — e.g. the port was already in use.
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining in-flight requests",
			slog.Duration("timeout", cfg.ShutdownTimeout))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed; forcing close", slog.String("error", err.Error()))
		_ = srv.Close()
		return err
	}

	logger.Info("server stopped cleanly")
	return <-serveErr // surface any late serve error (usually nil)
}
