package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/tapclap/mysql-master-health-checker/internal/mysql"
)

// MySQLInspector inspects MySQL availability and read_only state.
type MySQLInspector interface {
	Inspect(ctx context.Context) mysql.Status
}

// Runner periodically checks MySQL and updates the shared store.
type Runner struct {
	inspector MySQLInspector
	store     *Store
	interval  time.Duration
	logger    *slog.Logger
}

// NewRunner creates a background health checker.
func NewRunner(inspector MySQLInspector, store *Store, interval time.Duration, logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{
		inspector: inspector,
		store:     store,
		interval:  interval,
		logger:    logger,
	}
}

// Run executes checks until the context is canceled.
func (r *Runner) Run(ctx context.Context) {
	r.checkOnce(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkOnce(ctx)
		}
	}
}

func (r *Runner) checkOnce(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	status := r.inspector.Inspect(checkCtx)
	result := Result{
		Available:     status.Available,
		ReadOnly:      status.ReadOnly,
		ReadOnlyKnown: status.ReadOnlyKnown,
		CheckedAt:     time.Now().UTC(),
		Healthy:       status.Available && status.ReadOnlyKnown && !status.ReadOnly,
	}

	switch {
	case !status.Available:
		result.Reason = "mysql is unavailable"
		r.logger.Warn("mysql health check failed", "reason", result.Reason)
	case !status.ReadOnlyKnown:
		result.Reason = "mysql read_only state is unknown"
		r.logger.Warn("mysql health check failed", "reason", result.Reason)
	case status.ReadOnly:
		result.Reason = "mysql read_only is enabled"
		r.logger.Warn("mysql health check failed", "reason", result.Reason)
	default:
		result.Reason = "mysql is healthy"
		r.logger.Debug("mysql health check succeeded")
	}

	r.store.Update(result)
}
