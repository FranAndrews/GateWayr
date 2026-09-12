// Package proxy wraps net/http/httputil.ReverseProxy with load-balanced
// backend selection and custom error handling.
package proxy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"

	"github.com/FranAndrews/GateWayr/internal/loadbalancer"
)

// New creates an http.Handler that reverse-proxies requests to the next
// healthy backend selected by the load balancer.
//
// The returned handler:
//   - Rewrites the request URL scheme, host, and path to the target backend.
//   - Sets forwarding headers (X-Forwarded-For, X-Real-IP, X-Forwarded-Host).
//   - Returns a JSON 502 error on backend failure.
func New(lb *loadbalancer.RoundRobin, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, err := lb.NextBackend()
		if err != nil {
			logger.Error("no healthy backends available", slog.String("error", err.Error()))
			writeJSONError(w, http.StatusBadGateway, "bad_gateway", "all backend servers are unavailable")
			return
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.Host = target.Host

				// Preserve the original path — the backend handles routing.
				// Only rewrite scheme + host.

				// Set standard forwarding headers.
				if clientIP := req.Header.Get("X-Real-IP"); clientIP == "" {
					req.Header.Set("X-Real-IP", req.RemoteAddr)
				}
				req.Header.Set("X-Forwarded-Host", req.Host)
				if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
					req.Header.Set("X-Forwarded-For", prior+", "+req.RemoteAddr)
				} else {
					req.Header.Set("X-Forwarded-For", req.RemoteAddr)
				}

				logger.Debug("proxying request",
					slog.String("method", req.Method),
					slog.String("path", req.URL.Path),
					slog.String("target", target.String()),
				)
			},
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, e error) {
				logger.Error("proxy error",
					slog.String("target", target.String()),
					slog.String("error", e.Error()),
					slog.String("path", req.URL.Path),
				)
				// Mark the failing backend as down for future requests.
				lb.MarkDown(target)
				writeJSONError(rw, http.StatusBadGateway, "bad_gateway",
					fmt.Sprintf("upstream server %s is unavailable", target.Host))
			},
			ModifyResponse: func(resp *http.Response) error {
				logger.Info("upstream response",
					slog.String("target", target.String()),
					slog.Int("status", resp.StatusCode),
					slog.String("path", r.URL.Path),
				)
				return nil
			},
		}

		proxy.ServeHTTP(w, r)
	})
}

func writeJSONError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   errType,
		"code":    status,
		"message": message,
	})
}
