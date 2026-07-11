//go:build integration

package health

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/tapclap/mysql-master-health-checker/internal/mysql"
	"github.com/tapclap/mysql-master-health-checker/internal/testutil/mysqlcontainer"
)

func TestRunnerUpdatesStoreFromRealMySQL(t *testing.T) {
	checker, err := mysql.Open(mysqlcontainer.Start(t))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = checker.Close()
	}()

	store := NewStore()
	runner := NewRunner(checker, store, time.Hour, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runner.Run(ctx)

	mysqlcontainer.WaitForHealthyStore(t, 30*time.Second, func() bool {
		snapshot := store.Snapshot()
		return snapshot.Available && snapshot.ReadOnlyKnown && !snapshot.ReadOnly && snapshot.Reason == "mysql is healthy"
	})
}

func TestEvaluateHealthyWithRealMySQL(t *testing.T) {
	checker, err := mysql.Open(mysqlcontainer.Start(t))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = checker.Close()
	}()

	store := NewStore()
	runner := NewRunner(checker, store, time.Hour, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runner.Run(ctx)
	t.Setenv("MYSQL_MASTER", "true")

	mysqlcontainer.WaitForHealthyStore(t, 30*time.Second, func() bool {
		return Evaluate(store).Healthy
	})
}

func TestEvaluateUnhealthyWhenMasterEnvDisabled(t *testing.T) {
	checker, err := mysql.Open(mysqlcontainer.Start(t))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = checker.Close()
	}()

	store := NewStore()
	runner := NewRunner(checker, store, time.Hour, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runner.Run(ctx)
	t.Setenv("MYSQL_MASTER", "0")

	mysqlcontainer.WaitForHealthyStore(t, 30*time.Second, func() bool {
		snapshot := store.Snapshot()
		if !snapshot.Available || !snapshot.ReadOnlyKnown {
			return false
		}
		eval := Evaluate(store)
		return !eval.Healthy && eval.FailureReason == FailureReasonMasterEnv
	})
}

func TestEvaluateUnhealthyWhenReadOnlyEnabled(t *testing.T) {
	dsn := mysqlcontainer.Start(t)
	mysqlcontainer.SetGlobalReadOnly(t, dsn, true)

	checker, err := mysql.Open(dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = checker.Close()
	}()

	store := NewStore()
	runner := NewRunner(checker, store, time.Hour, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runner.Run(ctx)
	t.Setenv("MYSQL_MASTER", "1")

	mysqlcontainer.WaitForHealthyStore(t, 30*time.Second, func() bool {
		return Evaluate(store).FailureReason == FailureReasonReadOnly
	})
}
