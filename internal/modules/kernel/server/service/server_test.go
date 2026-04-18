package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalOnlyHostGuardAllowsLoopbackHost(t *testing.T) {
	handler := localOnlyHostGuard(okHandler(), "127.0.0.1")
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLocalOnlyHostGuardRejectsNonLoopbackHost(t *testing.T) {
	handler := localOnlyHostGuard(okHandler(), "127.0.0.1")
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestLocalOnlyHostGuardKeepsExplicitNonLoopbackBindCompatible(t *testing.T) {
	handler := localOnlyHostGuard(okHandler(), "0.0.0.0")
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
