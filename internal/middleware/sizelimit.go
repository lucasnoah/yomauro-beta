package middleware

import (
	"errors"
	"net/http"
	"sync"
)

// MaxRequestBody returns middleware that limits the size of incoming request
// bodies to the specified number of bytes. The limit is enforced using
// http.MaxBytesReader, which returns a *http.MaxBytesError on Read once the
// threshold is crossed. Handlers must inspect the read error and respond with
// 413 Request Entity Too Large themselves; this middleware does not send any
// response automatically.
func MaxRequestBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxResponseBody returns middleware that limits the size of outgoing response
// bodies to the specified number of bytes. Once the cumulative bytes written
// exceed the limit, the middleware panics with http.ErrAbortHandler. No bytes
// from the oversized write are flushed; only bytes from writes that fully fit
// within the remaining budget are sent. The panic propagates to the server,
// which closes the connection.
func MaxResponseBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &limitedResponseWriter{
				ResponseWriter: w,
				remaining:      limit,
			}
			next.ServeHTTP(lw, r)
		})
	}
}

// limitedResponseWriter wraps an http.ResponseWriter and enforces a maximum
// number of bytes written to the response body.
type limitedResponseWriter struct {
	http.ResponseWriter
	remaining   int64
	exceeded    bool
	wroteHeader bool
	mu          sync.Mutex
}

func (w *limitedResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	if !w.wroteHeader {
		w.wroteHeader = true
		w.mu.Unlock()
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.mu.Unlock()
}

func (w *limitedResponseWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	if w.exceeded {
		w.mu.Unlock()
		return 0, errors.New("response size limit exceeded")
	}

	n := int64(len(b))
	if n <= w.remaining {
		w.remaining -= n
		w.mu.Unlock()
		return w.ResponseWriter.Write(b)
	}

	// The write would exceed the limit. Mark as exceeded and abort without
	// flushing any bytes from this call — a partial write would produce a
	// malformed response body for the client.
	w.remaining = 0
	w.exceeded = true
	w.mu.Unlock()

	panic(http.ErrAbortHandler)
}
