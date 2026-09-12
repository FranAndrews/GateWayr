// Package loadbalancer implements a thread-safe round-robin load balancer
// that distributes requests across a pool of backend servers.
package loadbalancer

import (
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
)

// Backend represents a single upstream server instance.
type Backend struct {
	URL   *url.URL
	Alive bool
}

// RoundRobin distributes traffic evenly across backends using an atomic counter.
// Dead backends are skipped. If all backends are down, NextBackend returns an error.
type RoundRobin struct {
	backends []*Backend
	mu       sync.RWMutex
	current  uint64
}

// New creates a RoundRobin balancer from a list of backend URL strings.
func New(urls []string) (*RoundRobin, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("loadbalancer: at least one backend URL is required")
	}

	backends := make([]*Backend, 0, len(urls))
	for _, rawURL := range urls {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("loadbalancer: invalid backend URL %q: %w", rawURL, err)
		}
		backends = append(backends, &Backend{URL: u, Alive: true})
	}

	return &RoundRobin{backends: backends}, nil
}

// NextBackend returns the URL of the next healthy backend in round-robin order.
// It will cycle through all backends at most once to find a live one.
func (rr *RoundRobin) NextBackend() (*url.URL, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	total := uint64(len(rr.backends))
	if total == 0 {
		return nil, fmt.Errorf("loadbalancer: no backends configured")
	}

	// Try each backend at most once.
	for i := uint64(0); i < total; i++ {
		idx := atomic.AddUint64(&rr.current, 1) % total
		backend := rr.backends[idx]
		if backend.Alive {
			return backend.URL, nil
		}
	}

	return nil, fmt.Errorf("loadbalancer: all backends are down")
}

// MarkDown marks the backend with the given URL as unhealthy.
func (rr *RoundRobin) MarkDown(target *url.URL) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for _, b := range rr.backends {
		if b.URL.String() == target.String() {
			b.Alive = false
			return
		}
	}
}

// MarkUp marks the backend with the given URL as healthy.
func (rr *RoundRobin) MarkUp(target *url.URL) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for _, b := range rr.backends {
		if b.URL.String() == target.String() {
			b.Alive = true
			return
		}
	}
}

// Backends returns a snapshot of all backends and their health status.
// Useful for diagnostics and health-check endpoints.
func (rr *RoundRobin) Backends() []Backend {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	snapshot := make([]Backend, len(rr.backends))
	for i, b := range rr.backends {
		snapshot[i] = Backend{URL: b.URL, Alive: b.Alive}
	}
	return snapshot
}
