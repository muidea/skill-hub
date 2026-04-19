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

func TestLocalOnlyBrowserGuardAllowsCLIBridgeWriteWithoutOrigin(t *testing.T) {
	handler := localOnlyBrowserGuard(okHandler(), "127.0.0.1")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/repos/main/sync", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLocalOnlyBrowserGuardAllowsLoopbackOriginWrite(t *testing.T) {
	handler := localOnlyBrowserGuard(okHandler(), "127.0.0.1")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/repos/main/sync", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5525")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLocalOnlyBrowserGuardRejectsCrossSiteWrite(t *testing.T) {
	handler := localOnlyBrowserGuard(okHandler(), "127.0.0.1")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/repos/main/sync", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestLocalOnlyBrowserGuardRejectsNullOriginWrite(t *testing.T) {
	handler := localOnlyBrowserGuard(okHandler(), "127.0.0.1")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/repos/main/sync", nil)
	req.Header.Set("Origin", "null")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestLocalOnlyBrowserGuardRejectsCrossSiteFetchMetadata(t *testing.T) {
	handler := localOnlyBrowserGuard(okHandler(), "127.0.0.1")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/repos/main/sync", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestLocalOnlyBrowserGuardKeepsExplicitNonLoopbackBindCompatible(t *testing.T) {
	handler := localOnlyBrowserGuard(okHandler(), "0.0.0.0")
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/repos/main/sync", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(okHandler())
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing frame guard header")
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("missing content security policy header")
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
