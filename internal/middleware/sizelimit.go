package middleware

import (
	"errors"
	"net/http"
	"sync"
)

// MaxRequestBody returns middleware that limits the size of incoming request
// bodies to the specified number of bytes. Requests that exceed the limit
// receive a 413 Request Entity Too Large response. The limit is enforced
// using http.MaxBytesReader, which returns an error on Read after the
// threshold is crossed.
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
// bodies to the specified number of bytes. Once the limit is exceeded, further
// writes are silently discarded and the connection is closed via panic with
// http.ErrAbortHandler to prevent sending a partial, oversized response.
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

	// Write only the portion that fits within the limit.
	allowed := w.remaining
	w.remaining = 0
	w.exceeded = true
	w.mu.Unlock()

	if allowed > 0 {
		_, _ = w.ResponseWriter.Write(b[:allowed])
	}

	// Abort the response — the handler has produced more data than allowed.
	panic(http.ErrAbortHandler)
}
