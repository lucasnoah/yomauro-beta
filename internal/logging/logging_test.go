package logging

import (
	"context"
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
	logger := New("production")
	if _, ok := logger.Handler().(*slog.JSONHandler); !ok {
		t.Errorf("production logger should use JSONHandler, got %T", logger.Handler())
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
