package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- MaxRequestBody tests ---

func TestMaxRequestBody_AllowsWithinLimit(t *testing.T) {
	body := strings.NewReader("hello")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()

	var readErr error
	handler := MaxRequestBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	if readErr != nil {
		t.Fatalf("expected no error reading body within limit, got %v", readErr)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMaxRequestBody_RejectsOverLimit(t *testing.T) {
	body := strings.NewReader(strings.Repeat("x", 100))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()

	var readErr error
	handler := MaxRequestBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	if readErr == nil {
		t.Fatal("expected error reading body over limit")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rec.Code)
	}
}

func TestMaxRequestBody_NilBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := MaxRequestBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for nil body, got %d", rec.Code)
	}
}

func TestMaxRequestBody_ExactLimit(t *testing.T) {
	body := strings.NewReader(strings.Repeat("x", 10))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()

	var readErr error
	handler := MaxRequestBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	if readErr != nil {
		t.Fatalf("expected no error reading body at exact limit, got %v", readErr)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// --- MaxResponseBody tests ---

func TestMaxResponseBody_AllowsWithinLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := MaxResponseBody(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("short response"))
	}))

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "short response" {
		t.Errorf("expected 'short response', got %q", rec.Body.String())
	}
}

func TestMaxResponseBody_AbortsOverLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := MaxResponseBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", 100)))
	}))

	// MaxResponseBody panics with http.ErrAbortHandler when the limit is exceeded.
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				if r == http.ErrAbortHandler {
					panicked = true
				} else {
					t.Fatalf("unexpected panic value: %v", r)
				}
			}
		}()
		handler.ServeHTTP(rec, req)
	}()

	if !panicked {
		t.Fatal("expected panic with http.ErrAbortHandler when response exceeds limit")
	}
}

func TestMaxResponseBody_ExactLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	data := strings.Repeat("x", 10)
	handler := MaxResponseBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(data))
	}))

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != data {
		t.Errorf("expected %q, got %q", data, rec.Body.String())
	}
}

func TestMaxResponseBody_MultipleWritesWithinLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := MaxResponseBody(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("first "))
		w.Write([]byte("second "))
		w.Write([]byte("third"))
	}))

	handler.ServeHTTP(rec, req)

	expected := "first second third"
	if rec.Body.String() != expected {
		t.Errorf("expected %q, got %q", expected, rec.Body.String())
	}
}

func TestMaxResponseBody_MultipleWritesExceedLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := MaxResponseBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("12345")) // 5 bytes, remaining = 5
		w.Write([]byte("67890")) // 5 bytes, remaining = 0
		w.Write([]byte("extra")) // exceeds — should panic
	}))

	panicked := false
	func() {
		defer func() {
			if r := recover(); r == http.ErrAbortHandler {
				panicked = true
			}
		}()
		handler.ServeHTTP(rec, req)
	}()

	if !panicked {
		t.Fatal("expected panic when cumulative writes exceed limit")
	}
}

func TestMaxResponseBody_WriteHeaderDelegatesOnce(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := MaxResponseBody(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.WriteHeader(http.StatusNotFound) // second call should be ignored
		w.Write([]byte("ok"))
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

// --- Default constants ---

func TestDefaultConstants(t *testing.T) {
	if DefaultMaxRequestBodySize != 1<<20 {
		t.Errorf("expected DefaultMaxRequestBodySize = 1MB, got %d", DefaultMaxRequestBodySize)
	}
	if DefaultMaxResponseBodySize != 5<<20 {
		t.Errorf("expected DefaultMaxResponseBodySize = 5MB, got %d", DefaultMaxResponseBodySize)
	}
}

// --- Integration: both middleware chained ---

func TestChainedMiddleware_RequestWithinLimit(t *testing.T) {
	body := bytes.NewReader([]byte("input"))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()

	handler := MaxRequestBody(100)(MaxResponseBody(100)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read error", http.StatusBadRequest)
				return
			}
			w.Write([]byte("echo: " + string(data)))
		}),
	))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "echo: input" {
		t.Errorf("expected 'echo: input', got %q", rec.Body.String())
	}
}
