// Package middleware provides HTTP middleware for the API Gateway.
// Each middleware function wraps an http.Handler, adding cross-cutting concerns
// like logging, authentication, and rate limiting to the request pipeline.
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder captures the status code written by downstream handlers
// so the logger middleware can report it.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Logger returns middleware that logs every request in structured JSON format
// using the stdlib slog package. Each log entry includes:
//   - HTTP method and path
//   - Response status code
//   - Request latency
//   - Client IP (remote address)
//   - X-Request-ID header if present
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			duration := time.Since(start)

			logger.Info("request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.statusCode),
				slog.String("duration", duration.String()),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("request_id", r.Header.Get("X-Request-ID")),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}
