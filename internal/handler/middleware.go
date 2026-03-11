package handler

import (
	"net/http"
	"strings"
)

// CORSMiddleware returns middleware that sets CORS headers for the given
// comma-separated list of allowed origins. Wildcard "*" is not supported
// because Access-Control-Allow-Credentials: true is required for cookie-based auth.
// Preflight OPTIONS requests from allowed origins receive a 204 No Content response.
func CORSMiddleware(allowedOrigins string) func(http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				// Always declare Vary: Origin so caches do not serve a
				// no-CORS-headers response to a later allowed-origin request.
				w.Header().Add("Vary", "Origin")

				if origins[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")

					if r.Method == http.MethodOptions {
						w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
						w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
						w.Header().Set("Access-Control-Max-Age", "86400")
						w.WriteHeader(http.StatusNoContent)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseOrigins splits a comma-separated origins string into a set for O(1) lookup.
func parseOrigins(raw string) map[string]bool {
	origins := make(map[string]bool)
	for o := range strings.SplitSeq(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = true
		}
	}
	return origins
}
