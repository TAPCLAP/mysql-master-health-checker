package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	defaultListenAddr        = ":18848"
	defaultMySQLHost         = "127.0.0.1"
	defaultMySQLPort         = "3306"
	defaultCheckInterval     = 5 * time.Second
	defaultMetricsListenAddr = "127.0.0.1:18849"
)

// Config holds runtime configuration for the health check service.
type Config struct {
	ListenAddr    string
	TLSEnabled    bool
	TLSCertFile   string
	TLSKeyFile    string
	MySQLDSN      string
	CheckInterval time.Duration
	Metrics       MetricsConfig
}

// MetricsConfig holds Prometheus exporter settings.
type MetricsConfig struct {
	Enabled     bool
	ListenAddr  string
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
}

// Load reads configuration from command-line flags and environment variables.
func Load() (Config, error) {
	return load(flag.CommandLine, os.Args[1:])
}

func load(fs *flag.FlagSet, args []string) (Config, error) {
	listen := fs.String("listen", defaultListenAddr, "health-check listen address")
	tlsEnabled := fs.Bool("tls-enabled", true, "enable TLS for health-check endpoint")
	tlsCert := fs.String("tls-cert", "", "path to TLS certificate file")
	tlsKey := fs.String("tls-key", "", "path to TLS private key file")
	mysqlHost := fs.String("mysql-host", defaultMySQLHost, "MySQL host")
	mysqlPort := fs.String("mysql-port", defaultMySQLPort, "MySQL port")
	mysqlUser := fs.String("mysql-user", "", "MySQL user")
	mysqlPassword := fs.String("mysql-password", "", "MySQL password")
	mysqlDSN := fs.String("mysql-dsn", "", "MySQL DSN (overrides host/port/user/password)")
	checkInterval := fs.Duration("check-interval", defaultCheckInterval, "MySQL check interval")

	metricsEnabled := fs.Bool("metrics-enabled", true, "enable Prometheus metrics exporter")
	metricsListen := fs.String("metrics-listen", defaultMetricsListenAddr, "Prometheus metrics listen address")
	metricsTLSEnabled := fs.Bool("metrics-tls-enabled", false, "enable TLS for Prometheus metrics endpoint")
	metricsTLSCert := fs.String("metrics-tls-cert", "", "path to TLS certificate file for metrics endpoint")
	metricsTLSKey := fs.String("metrics-tls-key", "", "path to TLS private key file for metrics endpoint")

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	cfg := Config{
		ListenAddr:    resolveString("LISTEN_ADDR", *listen, defaultListenAddr),
		TLSEnabled:    resolveBool("TLS_ENABLED", *tlsEnabled, true),
		TLSCertFile:   resolveString("TLS_CERT_FILE", *tlsCert, ""),
		TLSKeyFile:    resolveString("TLS_KEY_FILE", *tlsKey, ""),
		CheckInterval: resolveDuration("CHECK_INTERVAL", *checkInterval, defaultCheckInterval),
		Metrics: MetricsConfig{
			Enabled:     resolveBool("METRICS_ENABLED", *metricsEnabled, true),
			ListenAddr:  resolveString("METRICS_LISTEN_ADDR", *metricsListen, defaultMetricsListenAddr),
			TLSEnabled:  resolveBool("METRICS_TLS_ENABLED", *metricsTLSEnabled, false),
			TLSCertFile: resolveString("METRICS_TLS_CERT_FILE", *metricsTLSCert, ""),
			TLSKeyFile:  resolveString("METRICS_TLS_KEY_FILE", *metricsTLSKey, ""),
		},
	}

	if dsn := resolveString("MYSQL_DSN", *mysqlDSN, ""); dsn != "" {
		cfg.MySQLDSN = dsn
	} else {
		user := resolveString("MYSQL_USER", *mysqlUser, "")
		if user == "" {
			return Config{}, errors.New("MYSQL_USER or --mysql-user is required when MYSQL_DSN is not set")
		}
		password := resolveString("MYSQL_PASSWORD", *mysqlPassword, "")
		host := resolveString("MYSQL_HOST", *mysqlHost, defaultMySQLHost)
		port := resolveString("MYSQL_PORT", *mysqlPort, defaultMySQLPort)
		cfg.MySQLDSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/?timeout=3s&readTimeout=3s&writeTimeout=3s&parseTime=true",
			user, password, host, port)
	}

	if cfg.TLSEnabled {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			return Config{}, errors.New("TLS_CERT_FILE and TLS_KEY_FILE (or --tls-cert and --tls-key) are required when TLS is enabled")
		}
	}
	if cfg.CheckInterval <= 0 {
		return Config{}, errors.New("check interval must be positive")
	}
	if cfg.Metrics.Enabled && cfg.Metrics.TLSEnabled {
		if cfg.Metrics.TLSCertFile == "" || cfg.Metrics.TLSKeyFile == "" {
			return Config{}, errors.New("METRICS_TLS_CERT_FILE and METRICS_TLS_KEY_FILE are required when metrics TLS is enabled")
		}
	}

	return cfg, nil
}
