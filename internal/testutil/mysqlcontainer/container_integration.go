//go:build integration

package mysqlcontainer

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

// Start boots a disposable MySQL instance and returns its DSN.
func Start(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	container, err := tcmysql.Run(ctx,
		"mysql:8.4",
		tcmysql.WithUsername("root"),
		tcmysql.WithPassword("secret"),
		tcmysql.WithDatabase("healthcheck"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}

	t.Cleanup(func() {
		terminateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(terminateCtx); err != nil {
			t.Logf("terminate mysql container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "timeout=5s", "readTimeout=5s", "writeTimeout=5s")
	if err != nil {
		t.Fatalf("mysql connection string: %v", err)
	}

	return dsn
}

// SetGlobalReadOnly toggles @@global.read_only on the given instance.
func SetGlobalReadOnly(t *testing.T, dsn string, enabled bool) {
	t.Helper()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql for read_only update: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	value := 0
	if enabled {
		value = 1
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("SET GLOBAL read_only = %d", value)); err != nil {
		t.Fatalf("set global read_only = %d: %v", value, err)
	}
}

// WaitForHealthyStore polls until fn returns true or timeout expires.
func WaitForHealthyStore(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("condition was not met before timeout")
}
