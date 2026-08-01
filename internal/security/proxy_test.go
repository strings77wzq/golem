package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Regression tests for the X-Forwarded-For trust boundary (security review
// H1): proxy headers must not drive rate limiting unless the direct peer is
// explicitly trusted.

func TestRateLimitIgnoresSpoofedXFF(t *testing.T) {
	cfg := RateLimitConfig{Rate: 1, Burst: 1, Enabled: true} // no trusted proxies
	mw, cleanup := RateLimitMiddlewareWithCleanup(cfg)
	defer cleanup()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Two requests from the same direct peer with different spoofed XFF
	// headers must share one bucket: the second is limited.
	for i, xff := range []string{"1.1.1.1", "2.2.2.2"} {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "203.0.113.9:4321"
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		want := http.StatusOK
		if i == 1 {
			want = http.StatusTooManyRequests
		}
		if rec.Code != want {
			t.Errorf("request %d: expected %d, got %d", i, want, rec.Code)
		}
	}
}

func TestRateLimitTrustsXFFFromTrustedProxy(t *testing.T) {
	cfg := RateLimitConfig{Rate: 1, Burst: 1, Enabled: true, TrustedProxies: []string{"127.0.0.1"}}
	mw, cleanup := RateLimitMiddlewareWithCleanup(cfg)
	defer cleanup()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Direct peer is trusted, so distinct XFF values map to distinct buckets.
	for _, xff := range []string{"1.1.1.1", "2.2.2.2"} {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:9999"
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("XFF %s: expected 200, got %d", xff, rec.Code)
		}
	}
}

func TestAllowFromIgnoresSpoofedXFF(t *testing.T) {
	// AllowFrom checks must also use the direct peer when no proxy is trusted.
	cfg := AuthConfig{
		Enabled:   true,
		APIKeys:   []string{"secret"},
		AllowFrom: []string{"203.0.113.9/32"}, // only the direct peer
	}
	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.9:4321"
	req.Header.Set("X-Forwarded-For", "9.9.9.9") // spoofed, must be ignored
	req.Header.Set("X-API-Key", "secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (direct peer allowed), got %d", rec.Code)
	}

	// A direct peer outside the allowlist is rejected even with a spoofed
	// XFF claiming the allowed IP.
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "198.51.100.7:4321"
	req2.Header.Set("X-Forwarded-For", "203.0.113.9")
	req2.Header.Set("X-API-Key", "secret")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Errorf("expected 403 (direct peer not allowed), got %d", rec2.Code)
	}
}
