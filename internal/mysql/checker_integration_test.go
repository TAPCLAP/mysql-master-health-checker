//go:build integration

package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/tapclap/mysql-master-health-checker/internal/testutil/mysqlcontainer"
)

func TestCheckerInspectHealthyMySQL(t *testing.T) {
	dsn := mysqlcontainer.Start(t)

	checker, err := Open(dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = checker.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := checker.Inspect(ctx)
	if !status.Available {
		t.Fatalf("Inspect() Available = false, want true")
	}
	if !status.ReadOnlyKnown {
		t.Fatalf("Inspect() ReadOnlyKnown = false, want true")
	}
	if status.ReadOnly {
		t.Fatalf("Inspect() ReadOnly = true, want false")
	}
	if err := checker.Check(ctx); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestCheckerInspectReadOnlyMySQL(t *testing.T) {
	dsn := mysqlcontainer.Start(t)
	mysqlcontainer.SetGlobalReadOnly(t, dsn, true)

	checker, err := Open(dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = checker.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := checker.Inspect(ctx)
	if !status.Available {
		t.Fatalf("Inspect() Available = false, want true")
	}
	if !status.ReadOnlyKnown {
		t.Fatalf("Inspect() ReadOnlyKnown = false, want true")
	}
	if !status.ReadOnly {
		t.Fatalf("Inspect() ReadOnly = false, want true")
	}
	if err := checker.Check(ctx); err == nil {
		t.Fatal("Check() error = nil, want read_only error")
	}
}

func TestCheckerInspectUnavailableMySQL(t *testing.T) {
	checker, err := Open("health:secret@tcp(127.0.0.1:1)/healthcheck?timeout=1s&readTimeout=1s&writeTimeout=1s")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = checker.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := checker.Inspect(ctx)
	if status.Available {
		t.Fatal("Inspect() Available = true, want false")
	}
	if status.ReadOnlyKnown {
		t.Fatal("Inspect() ReadOnlyKnown = true, want false")
	}
}
