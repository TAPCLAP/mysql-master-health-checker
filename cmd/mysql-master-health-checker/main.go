package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tapclap/mysql-master-health-checker/internal/config"
	"github.com/tapclap/mysql-master-health-checker/internal/health"
	"github.com/tapclap/mysql-master-health-checker/internal/metrics"
	"github.com/tapclap/mysql-master-health-checker/internal/mysql"
	"github.com/tapclap/mysql-master-health-checker/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	checker, err := mysql.Open(cfg.MySQLDSN)
	if err != nil {
		logger.Error("failed to open mysql", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := checker.Close(); err != nil {
			logger.Warn("failed to close mysql checker", "error", err)
		}
	}()

	store := health.NewStore()
	runner := health.NewRunner(checker, store, cfg.CheckInterval, logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go runner.Run(ctx)

	srv := server.New(cfg.ListenAddr, cfg.TLSCertFile, cfg.TLSKeyFile, store, logger)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServeTLS(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health server stopped with error", "error", err)
			cancel()
		}
	}()

	if cfg.Metrics.Enabled {
		exporter := metrics.New(store, cfg.Metrics, logger)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := exporter.Start(ctx); err != nil {
				logger.Error("metrics server stopped with error", "error", err)
				cancel()
			}
		}()
	}

	<-ctx.Done()
	logger.Info("shutdown requested")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown failed", "error", err)
		os.Exit(1)
	}

	wg.Wait()
}
