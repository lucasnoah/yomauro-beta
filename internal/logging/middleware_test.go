package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	var capturedID string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestID(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if capturedID == "" {
		t.Error("expected request ID in context")
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID response header")
	}
	if rec.Header().Get("X-Request-ID") != capturedID {
		t.Error("response header and context request ID should match")
	}
}

func TestRequestIDMiddleware_ReusesIncoming(t *testing.T) {
	var capturedID string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestID(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "incoming-id-42")
	handler.ServeHTTP(rec, req)

	if capturedID != "incoming-id-42" {
		t.Errorf("expected 'incoming-id-42', got %q", capturedID)
	}
	if rec.Header().Get("X-Request-ID") != "incoming-id-42" {
		t.Errorf("expected response header 'incoming-id-42', got %q", rec.Header().Get("X-Request-ID"))
	}
}

func TestLoggingMiddleware_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	ctx := WithRequestID(req.Context(), "test-req-id")
	req = req.WithContext(ctx)

	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["msg"] != "http request" {
		t.Errorf("expected msg 'http request', got %v", entry["msg"])
	}
	if entry["method"] != "POST" {
		t.Errorf("expected method POST, got %v", entry["method"])
	}
	if entry["path"] != "/api/test" {
		t.Errorf("expected path '/api/test', got %v", entry["path"])
	}
	// Status is stored as float64 in JSON unmarshalling.
	if status, ok := entry["status"].(float64); !ok || int(status) != 201 {
		t.Errorf("expected status 201, got %v", entry["status"])
	}
	if entry["request_id"] != "test-req-id" {
		t.Errorf("expected request_id 'test-req-id', got %v", entry["request_id"])
	}
	if _, ok := entry["duration"]; !ok {
		t.Error("expected duration in log entry")
	}
}

func TestLoggingMiddleware_DefaultStatus200(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}
	if status, ok := entry["status"].(float64); !ok || int(status) != 200 {
		t.Errorf("expected default status 200, got %v", entry["status"])
	}
}

func TestStatusWriter_WriteHeaderOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, status: http.StatusOK}

	sw.WriteHeader(http.StatusNotFound)
	sw.WriteHeader(http.StatusInternalServerError) // should be ignored

	if sw.status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", sw.status)
	}
}
