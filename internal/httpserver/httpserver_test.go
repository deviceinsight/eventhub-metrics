package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthAlwaysAvailable(t *testing.T) {
	s := NewServer(":0", time.Second)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) != "OK\n" {
		t.Fatalf("expected body %q, got %q", "OK\n", string(body))
	}
}

func TestHandleMountsAdditionalHandler(t *testing.T) {
	s := NewServer(":0", time.Second)
	s.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("metrics"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// /health must remain available alongside the mounted handler.
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /health status %d, got %d", http.StatusOK, rec.Code)
	}
}
