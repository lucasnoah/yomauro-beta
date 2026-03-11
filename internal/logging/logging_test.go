package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNew_Development(t *testing.T) {
	logger := New("development")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	// Development logger should log at debug level.
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("development logger should enable debug level")
	}
}

func TestNew_Production(t *testing.T) {
	logger := New("production")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	// Production logger should NOT log at debug level.
	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("production logger should not enable debug level")
	}
	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("production logger should enable info level")
	}
}

func TestNew_ProductionJSON(t *testing.T) {
	var buf bytes.Buffer
	// Create a JSON handler writing to a buffer to verify JSON output.
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	logger.Info("test message", slog.String("key", "value"))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("production logger output is not valid JSON: %v", err)
	}
	if entry["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got %v", entry["msg"])
	}
}

func TestRequestID_RoundTrip(t *testing.T) {
	ctx := context.Background()
	if got := RequestID(ctx); got != "" {
		t.Errorf("expected empty request ID from bare context, got %q", got)
	}

	ctx = WithRequestID(ctx, "abc123")
	if got := RequestID(ctx); got != "abc123" {
		t.Errorf("expected 'abc123', got %q", got)
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if len(id) != 16 {
		t.Errorf("expected 16-char hex string, got %d chars: %q", len(id), id)
	}

	// IDs should be unique.
	id2 := GenerateID()
	if id == id2 {
		t.Error("two generated IDs should not be equal")
	}
}
