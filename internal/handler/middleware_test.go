package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	mw := CORSMiddleware("http://localhost:3000")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected Allow-Origin 'http://localhost:3000', got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Allow-Credentials 'true', got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("expected Vary 'Origin', got %q", got)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	mw := CORSMiddleware("http://localhost:3000")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Allow-Origin header for disallowed origin, got %q", got)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	mw := CORSMiddleware("http://localhost:3000")
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for preflight, got %d", rec.Code)
	}
	if called {
		t.Error("next handler should not be called for preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Allow-Methods header for preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("expected Allow-Headers header for preflight")
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("expected Max-Age '86400', got %q", got)
	}
}

func TestCORSMiddleware_PreflightDisallowedOrigin(t *testing.T) {
	mw := CORSMiddleware("http://localhost:3000")
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS headers for disallowed origin preflight, got %q", got)
	}
	if !called {
		t.Error("next handler should be called for non-CORS OPTIONS request")
	}
}

func TestCORSMiddleware_MultipleOrigins(t *testing.T) {
	mw := CORSMiddleware("http://localhost:3000, https://app.example.com")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First origin
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected first origin, got %q", got)
	}

	// Second origin
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected second origin, got %q", got)
	}
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	mw := CORSMiddleware("http://localhost:3000")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS headers when no Origin header, got %q", got)
	}
}

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]bool
	}{
		{"single", "http://localhost:3000", map[string]bool{"http://localhost:3000": true}},
		{"multiple", "http://a.com, http://b.com", map[string]bool{"http://a.com": true, "http://b.com": true}},
		{"empty string", "", map[string]bool{}},
		{"whitespace", "  http://a.com  ,  http://b.com  ", map[string]bool{"http://a.com": true, "http://b.com": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOrigins(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("expected %d origins, got %d", len(tt.expected), len(got))
			}
			for k := range tt.expected {
				if !got[k] {
					t.Errorf("expected origin %q", k)
				}
			}
		})
	}
}
