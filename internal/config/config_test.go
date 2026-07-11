package config

import (
	"flag"
	"testing"
	"time"
)

func testLoad(t *testing.T, args []string) (Config, error) {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	return load(fs, args)
}

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func TestLoadRequiresTLSWhenEnabled(t *testing.T) {
	clearEnv(t, "TLS_ENABLED", "TLS_CERT_FILE", "TLS_KEY_FILE", "MYSQL_DSN")
	t.Setenv("MYSQL_DSN", "user:pass@tcp(127.0.0.1:3306)/")

	_, err := testLoad(t, nil)
	if err == nil {
		t.Fatal("expected error when TLS files are missing")
	}
}

func TestLoadAllowsPlainHTTPWhenTLSDisabled(t *testing.T) {
	clearEnv(t, "TLS_ENABLED", "TLS_CERT_FILE", "TLS_KEY_FILE", "MYSQL_DSN")
	t.Setenv("TLS_ENABLED", "false")
	t.Setenv("MYSQL_DSN", "user:pass@tcp(127.0.0.1:3306)/")

	cfg, err := testLoad(t, nil)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if cfg.TLSEnabled {
		t.Fatal("TLS should be disabled")
	}
	if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" {
		t.Fatalf("unexpected TLS files: cert=%q key=%q", cfg.TLSCertFile, cfg.TLSKeyFile)
	}
}

func TestLoadBuildsDSNFromParts(t *testing.T) {
	clearEnv(t, "TLS_CERT_FILE", "TLS_KEY_FILE", "MYSQL_DSN", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_HOST", "MYSQL_PORT", "CHECK_INTERVAL")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")
	t.Setenv("MYSQL_USER", "health")
	t.Setenv("MYSQL_PASSWORD", "secret")
	t.Setenv("MYSQL_HOST", "db.local")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("CHECK_INTERVAL", "10s")

	cfg, err := testLoad(t, nil)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if cfg.MySQLDSN != "health:secret@tcp(db.local:3307)/?timeout=3s&readTimeout=3s&writeTimeout=3s&parseTime=true" {
		t.Fatalf("unexpected DSN: %q", cfg.MySQLDSN)
	}
	if cfg.CheckInterval != 10*time.Second {
		t.Fatalf("CheckInterval = %v, want 10s", cfg.CheckInterval)
	}
}

func TestLoadUsesExplicitDSN(t *testing.T) {
	clearEnv(t, "TLS_CERT_FILE", "TLS_KEY_FILE", "MYSQL_DSN")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")
	t.Setenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/mysql")

	cfg, err := testLoad(t, nil)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if cfg.MySQLDSN != "root:root@tcp(localhost:3306)/mysql" {
		t.Fatalf("unexpected DSN: %q", cfg.MySQLDSN)
	}
}

func TestLoadMetricsDefaults(t *testing.T) {
	clearEnv(t, "TLS_CERT_FILE", "TLS_KEY_FILE", "MYSQL_DSN", "METRICS_LISTEN_ADDR", "METRICS_TLS_ENABLED")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")
	t.Setenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/mysql")

	cfg, err := testLoad(t, nil)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if !cfg.Metrics.Enabled {
		t.Fatal("metrics should be enabled by default")
	}
	if cfg.Metrics.ListenAddr != defaultMetricsListenAddr {
		t.Fatalf("Metrics.ListenAddr = %q", cfg.Metrics.ListenAddr)
	}
	if cfg.Metrics.TLSEnabled {
		t.Fatal("metrics TLS should be disabled by default")
	}
}

func TestLoadMetricsTLSRequiresCertAndKey(t *testing.T) {
	clearEnv(t, "TLS_CERT_FILE", "TLS_KEY_FILE", "MYSQL_DSN", "METRICS_TLS_ENABLED", "METRICS_TLS_CERT_FILE", "METRICS_TLS_KEY_FILE")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")
	t.Setenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/mysql")
	t.Setenv("METRICS_TLS_ENABLED", "true")

	_, err := testLoad(t, nil)
	if err == nil {
		t.Fatal("expected error when metrics TLS is enabled without cert/key")
	}
}

func TestLoadMetricsFromFlags(t *testing.T) {
	clearEnv(t, "TLS_CERT_FILE", "TLS_KEY_FILE", "MYSQL_DSN", "METRICS_LISTEN_ADDR")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")
	t.Setenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/mysql")

	cfg, err := testLoad(t, []string{"--metrics-listen", "0.0.0.0:9100", "--metrics-enabled=false"})
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if cfg.Metrics.Enabled {
		t.Fatal("expected metrics to be disabled via flag")
	}
	if cfg.Metrics.ListenAddr != "0.0.0.0:9100" {
		t.Fatalf("Metrics.ListenAddr = %q", cfg.Metrics.ListenAddr)
	}
}
