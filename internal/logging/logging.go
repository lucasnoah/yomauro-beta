// Package logging provides structured slog logger configuration
// and HTTP middleware for request ID tracking and request logging.
package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// New creates a structured slog.Logger configured for the given environment.
// In "development" mode, outputs human-readable text at debug level.
// In all other modes, outputs JSON at info level.
func New(env string) *slog.Logger {
	if env == "development" {
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// RequestID extracts the request ID from the context.
// Returns an empty string if no request ID is present.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithRequestID returns a new context with the given request ID stored.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GenerateID returns a random 8-byte hex-encoded string (16 chars)
// suitable for use as a request ID.
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
