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

func TestMaxResponseBody_ZeroLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := MaxResponseBody(0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("x")) // any non-empty write must panic
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
		t.Fatal("expected panic with http.ErrAbortHandler for zero limit")
	}
	// No bytes from the oversized write must be in the response body.
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty response body, got %d bytes: %q", rec.Body.Len(), rec.Body.String())
	}
}

// TestMaxResponseBody_NoPartialWriteOnExceed verifies that when a single write
// exceeds the remaining budget, none of its bytes are flushed before the panic.
// Prior to the fix, the code wrote b[:allowed] before panicking, producing a
// malformed partial response body.
func TestMaxResponseBody_NoPartialWriteOnExceed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Limit of 5: the first write (3 bytes) fits; the second (10 bytes) exceeds.
	handler := MaxResponseBody(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("abc"))               // 3 bytes, fits
		w.Write([]byte("0123456789"))        // 10 bytes, exceeds — must not write any
	}))

	func() {
		defer func() { recover() }()
		handler.ServeHTTP(rec, req)
	}()

	// Only the first write's 3 bytes must appear in the response.
	if got := rec.Body.String(); got != "abc" {
		t.Errorf("expected body %q, got %q — partial bytes from oversized write must not be flushed", "abc", got)
	}
}

// TestMaxResponseBody_ExceededStateBlocksSubsequentWrites verifies that once
// the limit is exceeded (exceeded=true), any subsequent Write returns an error
// without panicking again or writing bytes. This path is reachable when a
// deferred function in the handler recovers the ErrAbortHandler panic and then
// writes to the response writer.
func TestMaxResponseBody_ExceededStateBlocksSubsequentWrites(t *testing.T) {
	lw := &limitedResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		remaining:      0,
		exceeded:       true,
	}

	n, err := lw.Write([]byte("after limit"))
	if err == nil {
		t.Fatal("expected error when writing after limit exceeded")
	}
	if n != 0 {
		t.Errorf("expected 0 bytes written, got %d", n)
	}
}

// TestMaxRequestBody_HandlerIgnoresReadError verifies that MaxRequestBody does
// NOT automatically send 413 when the handler reads the body but ignores the
// error. The 413 is the handler's responsibility.
func TestMaxRequestBody_HandlerIgnoresReadError(t *testing.T) {
	body := strings.NewReader(strings.Repeat("x", 100))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()

	handler := MaxRequestBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body) // intentionally ignoring the error
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	// Without the handler checking the error, the response is 200 — not 413.
	// This documents that MaxRequestBody alone does not enforce 413.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (handler did not check error), got %d", rec.Code)
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
