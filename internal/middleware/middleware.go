// Package middleware provides HTTP middleware for request and response
// size enforcement.
package middleware

// DefaultMaxRequestBodySize is the default maximum request body size (1 MB).
const DefaultMaxRequestBodySize int64 = 1 << 20

// DefaultMaxResponseBodySize is the default maximum response body size (5 MB).
const DefaultMaxResponseBodySize int64 = 5 << 20
