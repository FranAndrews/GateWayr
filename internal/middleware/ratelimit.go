package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter returns middleware that enforces a token-bucket rate limit
// per client IP address. When the bucket is exhausted, it returns
// 429 Too Many Requests with a JSON error body.
//
// Parameters:
//   - rps: sustained requests per second allowed per IP
//   - burst: maximum burst size (bucket capacity)
func RateLimiter(rps float64, burst int) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		limiters = make(map[string]*rate.Limiter)
	)

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		limiter, exists := limiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(rps), burst)
			limiters[ip] = limiter
		}
		return limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			limiter := getLimiter(ip)
			if !limiter.Allow() {
				writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
					"error":   "rate_limit_exceeded",
					"code":    http.StatusTooManyRequests,
					"message": "too many requests, please slow down",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
