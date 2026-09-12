package loadbalancer

import (
	"sync"
	"testing"
)

func TestNew_EmptyURLs(t *testing.T) {
	_, err := New([]string{})
	if err == nil {
		t.Fatal("expected error for empty URL list, got nil")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New([]string{"://bad-url"})
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestNextBackend_RoundRobinOrder(t *testing.T) {
	urls := []string{
		"http://backend-1:9000",
		"http://backend-2:9000",
		"http://backend-3:9000",
	}
	lb, err := New(urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The atomic counter starts at 0; AddUint64 pre-increments, so
	// the first call yields index (0+1)%3 = 1 → backend-2.
	// Subsequent calls cycle: backend-3, backend-1, backend-2, ...
	expected := []string{
		"http://backend-2:9000",
		"http://backend-3:9000",
		"http://backend-1:9000",
	}

	for cycle := 0; cycle < 3; cycle++ {
		for i, want := range expected {
			got, err := lb.NextBackend()
			if err != nil {
				t.Fatalf("cycle %d, index %d: unexpected error: %v", cycle, i, err)
			}
			if got.String() != want {
				t.Errorf("cycle %d, index %d: expected %s, got %s", cycle, i, want, got.String())
			}
		}
	}
}

func TestNextBackend_SkipsDeadBackends(t *testing.T) {
	urls := []string{
		"http://backend-1:9000",
		"http://backend-2:9000",
		"http://backend-3:9000",
	}
	lb, err := New(urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mark backend-2 as down (index 1). Advance once to consume index 0.
	got, _ := lb.NextBackend() // consumes index 1 → backend-2
	_ = got

	// Now mark backend-2 down so that subsequent calls skip it.
	lb.backends[1].Alive = false

	// Next call should skip the dead backend and return backend-3.
	got, err = lb.NextBackend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() != "http://backend-3:9000" {
		t.Errorf("expected backend-3, got %s", got.String())
	}
}

func TestNextBackend_AllDown(t *testing.T) {
	urls := []string{"http://backend-1:9000"}
	lb, err := New(urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lb.backends[0].Alive = false

	_, err = lb.NextBackend()
	if err == nil {
		t.Fatal("expected error when all backends are down, got nil")
	}
}

func TestMarkDownAndUp(t *testing.T) {
	urls := []string{"http://backend-1:9000", "http://backend-2:9000"}
	lb, err := New(urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	target := lb.backends[0].URL
	lb.MarkDown(target)
	if lb.backends[0].Alive {
		t.Error("expected backend to be marked down")
	}

	lb.MarkUp(target)
	if !lb.backends[0].Alive {
		t.Error("expected backend to be marked up")
	}
}

func TestNextBackend_ConcurrentAccess(t *testing.T) {
	urls := []string{
		"http://backend-1:9000",
		"http://backend-2:9000",
		"http://backend-3:9000",
	}
	lb, err := New(urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 100
	const callsPerGoroutine = 50

	errs := make(chan error, goroutines*callsPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := 0; c < callsPerGoroutine; c++ {
				_, err := lb.NextBackend()
				if err != nil {
					errs <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent call returned error: %v", err)
	}
}

func TestBackends_Snapshot(t *testing.T) {
	urls := []string{"http://backend-1:9000", "http://backend-2:9000"}
	lb, err := New(urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snap := lb.Backends()
	if len(snap) != 2 {
		t.Fatalf("expected 2 backends, got %d", len(snap))
	}
	// Mutating snapshot should not affect original.
	snap[0].Alive = false
	if !lb.backends[0].Alive {
		t.Error("snapshot mutation affected original backend list")
	}
}
