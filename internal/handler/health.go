package handler

import (
	"net/http"
	"os"
)

// HealthHandler returns a simple health-check handler that reports the
// instance name (from INSTANCE_NAME env var). This is critical for
// observing round-robin load balancing in action — each backend instance
// returns its own name, so consecutive calls show traffic distribution.
func HealthHandler() http.HandlerFunc {
	instanceName := os.Getenv("INSTANCE_NAME")
	if instanceName == "" {
		hostname, _ := os.Hostname()
		instanceName = hostname
	}

	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ok",
			"instance": instanceName,
		})
	}
}
