package ingest

import (
	"context"
	"log/slog"
	"time"
)

// Scheduler runs the worker on a fixed interval, decoupled from request handling (SPEC-006
// FR-604). It is the in-process option (toggled by MARKETDATA_SCHEDULER_ENABLED); the
// request-independent alternative is the cmd/ingest one-shot driven by cron. The ticker
// uses wall time for cadence; the worker stamps data via the injected Clock for determinism.
type Scheduler struct {
	worker   *Worker
	interval time.Duration
	logger   *slog.Logger
}

// NewScheduler returns a Scheduler that runs worker every interval.
func NewScheduler(worker *Worker, interval time.Duration, logger *slog.Logger) *Scheduler {
	return &Scheduler{worker: worker, interval: interval, logger: logger}
}

// Run executes one ingestion immediately (so data is fresh at startup), then every interval
// until ctx is canceled, at which point it returns. A run in flight when ctx is canceled
// finishes (its provider calls observe the canceled context and degrade).
func (s *Scheduler) Run(ctx context.Context) {
	s.worker.RunOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("market data scheduler stopped")
			return
		case <-ticker.C:
			s.worker.RunOnce(ctx)
		}
	}
}
