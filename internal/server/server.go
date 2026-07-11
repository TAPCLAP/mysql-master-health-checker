package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tapclap/mysql-master-health-checker/internal/health"
)

const (
	okBody       = "OK\n"
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
)

// Server exposes the cached health state over HTTP or HTTPS.
type Server struct {
	addr       string
	tlsEnabled bool
	certFile   string
	keyFile    string
	store      *health.Store
	logger     *slog.Logger
	httpServer *http.Server
}

// New creates a health-check HTTP server.
func New(addr string, tlsEnabled bool, certFile, keyFile string, store *health.Store, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		addr:       addr,
		tlsEnabled: tlsEnabled,
		certFile:   certFile,
		keyFile:    keyFile,
		store:      store,
		logger:     logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHealth)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
	}

	return s
}

// ListenAndServe starts the server and blocks until it stops.
func (s *Server) ListenAndServe() error {
	if s.tlsEnabled {
		s.logger.Info("starting health server with tls", "addr", s.addr)
		if err := s.httpServer.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("listen and serve tls: %w", err)
		}
		return nil
	}

	s.logger.Info("starting health server", "addr", s.addr, "tls", false)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	eval := health.Evaluate(s.store)
	if eval.Healthy {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(okBody))
		return
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(eval.Reason + "\n"))
}
