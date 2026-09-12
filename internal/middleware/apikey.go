package middleware

import (
	"encoding/json"
	"net/http"
	"slices"
)

// APIKeyAuth returns middleware that validates the X-API-Key header against
// a configured allowlist. Requests to paths in skipPaths bypass validation.
//
// On failure, returns 401 Unauthorized with a JSON error body.
func APIKeyAuth(validKeys []string, skipPaths []string) func(http.Handler) http.Handler {
	// Build a set for O(1) key lookups.
	keySet := make(map[string]struct{}, len(validKeys))
	for _, k := range validKeys {
		keySet[k] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for exempt paths (e.g. /health).
			if slices.Contains(skipPaths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
					"error":   "unauthorized",
					"code":    http.StatusUnauthorized,
					"message": "missing X-API-Key header",
				})
				return
			}

			if _, ok := keySet[apiKey]; !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
					"error":   "unauthorized",
					"code":    http.StatusUnauthorized,
					"message": "invalid API key",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
