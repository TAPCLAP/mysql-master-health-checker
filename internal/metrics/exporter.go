package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tapclap/mysql-master-health-checker/internal/config"
	"github.com/tapclap/mysql-master-health-checker/internal/health"
)

const metricsPath = "/metrics"

// Exporter serves Prometheus metrics for mysql-master-health-checker.
type Exporter struct {
	store    *health.Store
	logger   *slog.Logger
	cfg      config.MetricsConfig
	registry *prometheus.Registry

	mysqlAvailable    prometheus.Gauge
	readOnlyEnabled   prometheus.Gauge
	masterEnvEnabled  prometheus.Gauge
	healthy           prometheus.Gauge
	unhealthyReason   *prometheus.GaugeVec
	lastCheckUnixtime prometheus.Gauge

	mu         sync.Mutex
	lastReason health.FailureReason
	httpServer *http.Server
}

// New creates a Prometheus exporter backed by the shared health store.
func New(store *health.Store, cfg config.MetricsConfig, logger *slog.Logger) *Exporter {
	if logger == nil {
		logger = slog.Default()
	}

	e := &Exporter{
		store:      store,
		logger:     logger,
		cfg:        cfg,
		registry:   prometheus.NewRegistry(),
		lastReason: health.FailureReasonNone,
	}

	e.mysqlAvailable = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mysql_master_health_check_mysql_available",
		Help: "Whether MySQL is reachable (1=available, 0=unavailable).",
	})
	e.readOnlyEnabled = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mysql_master_health_check_mysql_read_only_enabled",
		Help: "Whether MySQL global read_only is enabled (1=enabled, 0=disabled). Set to -1 when read_only state is unknown.",
	})
	e.masterEnvEnabled = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mysql_master_health_check_master_env_enabled",
		Help: "Whether MYSQL_MASTER environment variable is enabled (1=enabled, 0=disabled).",
	})
	e.healthy = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mysql_master_health_check_ok",
		Help: "Whether all health checks pass and the service would return HTTP 200 OK (1=ok, 0=not ok).",
	})
	e.unhealthyReason = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_master_health_check_unhealthy_reason_info",
		Help: "Active reason why the service is not OK (value is always 1 for the active reason).",
	}, []string{"reason"})
	e.lastCheckUnixtime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mysql_master_health_check_last_mysql_check_timestamp",
		Help: "Unix timestamp of the last completed background MySQL check.",
	})

	e.registry.MustRegister(
		e.mysqlAvailable,
		e.readOnlyEnabled,
		e.masterEnvEnabled,
		e.healthy,
		e.unhealthyReason,
		e.lastCheckUnixtime,
	)

	return e
}

// Refresh updates gauge values from the current health state.
func (e *Exporter) Refresh() {
	eval := health.Evaluate(e.store)

	e.mysqlAvailable.Set(boolToFloat(eval.MySQLUp))

	switch {
	case !eval.ReadOnlyKnown:
		e.readOnlyEnabled.Set(-1)
	case eval.ReadOnly:
		e.readOnlyEnabled.Set(1)
	default:
		e.readOnlyEnabled.Set(0)
	}

	e.masterEnvEnabled.Set(boolToFloat(eval.Master))
	e.healthy.Set(boolToFloat(eval.Healthy))

	if !eval.MySQL.CheckedAt.IsZero() {
		e.lastCheckUnixtime.Set(float64(eval.MySQL.CheckedAt.Unix()))
	}

	e.setUnhealthyReason(eval.FailureReason)
}

func (e *Exporter) setUnhealthyReason(reason health.FailureReason) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.lastReason != "" && e.lastReason != reason {
		e.unhealthyReason.DeleteLabelValues(string(e.lastReason))
	}

	e.lastReason = reason
	if reason == health.FailureReasonNone {
		return
	}

	e.unhealthyReason.WithLabelValues(string(reason)).Set(1)
}

// Handler returns the Prometheus metrics HTTP handler.
func (e *Exporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{})
}

// Start runs the metrics HTTP server until the context is canceled.
func (e *Exporter) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.Refresh()
		e.Handler().ServeHTTP(w, r)
	}))

	e.httpServer = &http.Server{
		Addr:              e.cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if e.cfg.TLSEnabled {
			e.logger.Info("starting metrics server with tls", "addr", e.cfg.ListenAddr)
			errCh <- e.httpServer.ListenAndServeTLS(e.cfg.TLSCertFile, e.cfg.TLSKeyFile)
			return
		}
		e.logger.Info("starting metrics server", "addr", e.cfg.ListenAddr, "tls", false)
		errCh <- e.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("metrics shutdown: %w", err)
		}
		if err := <-errCh; err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("metrics server: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("metrics server: %w", err)
		}
		return nil
	}
}

func boolToFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
