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

// Server exposes the cached health state over HTTPS.
type Server struct {
	addr       string
	certFile   string
	keyFile    string
	store      *health.Store
	logger     *slog.Logger
	httpServer *http.Server
}

// New creates an HTTPS server.
func New(addr, certFile, keyFile string, store *health.Store, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		addr:     addr,
		certFile: certFile,
		keyFile:  keyFile,
		store:    store,
		logger:   logger,
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

// ListenAndServeTLS starts the HTTPS server and blocks until it stops.
func (s *Server) ListenAndServeTLS() error {
	s.logger.Info("starting https server", "addr", s.addr)
	if err := s.httpServer.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve tls: %w", err)
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
