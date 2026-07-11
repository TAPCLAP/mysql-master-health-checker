package health

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/tapclap/mysql-master-health-checker/internal/mysql"
)

type stubInspector struct {
	status mysql.Status
}

func (s stubInspector) Inspect(context.Context) mysql.Status {
	return s.status
}

func TestRunnerUpdatesStore(t *testing.T) {
	store := NewStore()
	runner := NewRunner(stubInspector{status: mysql.Status{
		Available: false,
	}}, store, time.Hour, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runner.Run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := store.Snapshot()
		if !snapshot.Available && snapshot.Reason == "mysql is unavailable" {
			cancel()
			<-done
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("store was not updated by runner")
}

func TestRunnerRecordsReadOnly(t *testing.T) {
	store := NewStore()
	runner := NewRunner(stubInspector{status: mysql.Status{
		Available:     true,
		ReadOnlyKnown: true,
		ReadOnly:      true,
	}}, store, time.Hour, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runner.Run(ctx)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := store.Snapshot()
		if snapshot.ReadOnly && snapshot.Reason == "mysql read_only is enabled" {
			cancel()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("read_only state was not stored")
}

var _ MySQLInspector = stubInspector{}
