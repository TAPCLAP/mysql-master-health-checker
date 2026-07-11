package server

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tapclap/mysql-master-health-checker/internal/health"
)

func TestHandleHealthOK(t *testing.T) {
	store := health.NewStore()
	store.Update(health.Result{
		Available:     true,
		ReadOnly:      false,
		ReadOnlyKnown: true,
		Healthy:       true,
		Reason:        "mysql is healthy",
		CheckedAt:     time.Now(),
	})
	t.Setenv("MYSQL_MASTER", "1")

	srv := New(":0", "", "", store, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != okBody {
		t.Fatalf("body = %q, want %q", body, okBody)
	}
}

func TestHandleHealthUnavailable(t *testing.T) {
	store := health.NewStore()
	store.Update(health.Result{
		Available: false,
		Healthy:   false,
		Reason:    "mysql ping: refused",
	})
	t.Setenv("MYSQL_MASTER", "1")

	srv := New(":0", "", "", store, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.handleHealth(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandleHealthRequiresMasterEnv(t *testing.T) {
	store := health.NewStore()
	store.Update(health.Result{
		Available:     true,
		ReadOnly:      false,
		ReadOnlyKnown: true,
		Healthy:       true,
	})
	t.Setenv("MYSQL_MASTER", "0")

	srv := New(":0", "", "", store, nil)
	rec := httptest.NewRecorder()
	srv.handleHealth(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestListenAndServeTLSUsesProvidedCertificate(t *testing.T) {
	certFile := "../../testdata/tls/server.crt"
	keyFile := "../../testdata/tls/server.key"

	store := health.NewStore()
	store.Update(health.Result{
		Available:     true,
		ReadOnly:      false,
		ReadOnlyKnown: true,
		Healthy:       true,
	})
	t.Setenv("MYSQL_MASTER", "1")

	srv := New("127.0.0.1:0", certFile, keyFile, store, nil)

	ts := httptest.NewUnstartedServer(http.HandlerFunc(srv.handleHealth))
	ts.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	ts.StartTLS()
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
