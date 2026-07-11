package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/tapclap/mysql-master-health-checker/internal/config"
	"github.com/tapclap/mysql-master-health-checker/internal/health"
)

func TestRefreshExportsIndividualChecks(t *testing.T) {
	store := health.NewStore()
	store.Update(health.Result{
		Available:     true,
		ReadOnly:      true,
		ReadOnlyKnown: true,
		Healthy:       false,
		CheckedAt:     time.Unix(1_700_000_000, 0),
	})
	t.Setenv("MYSQL_MASTER", "1")

	exporter := New(store, config.MetricsConfig{Enabled: true}, nil)
	exporter.Refresh()

	if err := testutil.CollectAndCompare(exporter.registry, strings.NewReader(`
# HELP mysql_master_health_check_mysql_available Whether MySQL is reachable (1=available, 0=unavailable).
# TYPE mysql_master_health_check_mysql_available gauge
mysql_master_health_check_mysql_available 1
# HELP mysql_master_health_check_mysql_read_only_enabled Whether MySQL global read_only is enabled (1=enabled, 0=disabled). Set to -1 when read_only state is unknown.
# TYPE mysql_master_health_check_mysql_read_only_enabled gauge
mysql_master_health_check_mysql_read_only_enabled 1
# HELP mysql_master_health_check_master_env_enabled Whether MYSQL_MASTER environment variable is enabled (1=enabled, 0=disabled).
# TYPE mysql_master_health_check_master_env_enabled gauge
mysql_master_health_check_master_env_enabled 1
# HELP mysql_master_health_check_ok Whether all health checks pass and the service would return HTTP 200 OK (1=ok, 0=not ok).
# TYPE mysql_master_health_check_ok gauge
mysql_master_health_check_ok 0
# HELP mysql_master_health_check_unhealthy_reason_info Active reason why the service is not OK (value is always 1 for the active reason).
# TYPE mysql_master_health_check_unhealthy_reason_info gauge
mysql_master_health_check_unhealthy_reason_info{reason="mysql_read_only"} 1
# HELP mysql_master_health_check_last_mysql_check_timestamp Unix timestamp of the last completed background MySQL check.
# TYPE mysql_master_health_check_last_mysql_check_timestamp gauge
mysql_master_health_check_last_mysql_check_timestamp 1.7e+09
`)); err != nil {
		t.Fatalf("CollectAndCompare() error = %v", err)
	}
}

func TestMetricsHandler(t *testing.T) {
	store := health.NewStore()
	store.Update(health.Result{
		Available:     true,
		ReadOnly:      false,
		ReadOnlyKnown: true,
		Healthy:       true,
		CheckedAt:     time.Now(),
	})
	t.Setenv("MYSQL_MASTER", "true")

	exporter := New(store, config.MetricsConfig{Enabled: true}, nil)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exporter.Refresh()
		exporter.Handler().ServeHTTP(w, r)
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "mysql_master_health_check_ok 1") {
		t.Fatalf("metrics body = %q", body)
	}
}

func TestStartStopsWithContext(t *testing.T) {
	store := health.NewStore()
	exporter := New(store, config.MetricsConfig{
		Enabled:    true,
		ListenAddr: "127.0.0.1:0",
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- exporter.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("metrics server did not stop")
	}
}
