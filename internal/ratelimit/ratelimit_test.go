package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu     sync.Mutex
	counts map[string]int
}

func (s *memoryStore) Consume(_ context.Context, windows []Window) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, window := range windows {
		if s.counts[window.Key] >= window.Limit {
			return ErrLimited
		}
	}
	for _, window := range windows {
		s.counts[window.Key]++
	}
	return nil
}

func testLimiter(now time.Time) (*Limiter, *memoryStore) {
	store := &memoryStore{counts: map[string]int{}}
	limiter := New(store, "test-secret")
	limiter.now = func() time.Time { return now }
	return limiter, store
}

func TestLimiterPermitsLimitsThenRejects(t *testing.T) {
	limiter, _ := testLimiter(time.Date(2026, 8, 22, 10, 10, 0, 0, time.UTC))
	for range MaxAcceptedJobsPerHour {
		if _, err := limiter.Allow(context.Background(), "203.0.113.8"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := limiter.Allow(context.Background(), "203.0.113.8"); err != ErrLimited {
		t.Fatalf("error = %v, want ErrLimited", err)
	}
}

func TestLimiterConcurrentRequestsCannotExceedLimit(t *testing.T) {
	limiter, _ := testLimiter(time.Date(2026, 8, 22, 10, 10, 0, 0, time.UTC))
	var accepted int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if _, err := limiter.Allow(context.Background(), "203.0.113.8"); err == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if accepted != MaxAcceptedJobsPerHour {
		t.Fatalf("accepted = %d, want %d", accepted, MaxAcceptedJobsPerHour)
	}
}

func TestLimiterUsesNewWindowAfterExpiry(t *testing.T) {
	limiter, _ := testLimiter(time.Date(2026, 8, 22, 10, 59, 0, 0, time.UTC))
	for range MaxAcceptedJobsPerHour {
		_, _ = limiter.Allow(context.Background(), "203.0.113.8")
	}
	limiter.now = func() time.Time { return time.Date(2026, 8, 22, 11, 0, 0, 0, time.UTC) }
	if _, err := limiter.Allow(context.Background(), "203.0.113.8"); err != nil {
		t.Fatalf("new window error = %v", err)
	}
}

func TestLimiterEnforcesDailyLimitAcrossHourlyWindows(t *testing.T) {
	limiter, _ := testLimiter(time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	for hour := 0; hour < 3; hour++ {
		limiter.now = func() time.Time { return time.Date(2026, 8, 22, hour, 0, 0, 0, time.UTC) }
		for range MaxAcceptedJobsPerHour {
			if _, err := limiter.Allow(context.Background(), "203.0.113.8"); err != nil {
				t.Fatal(err)
			}
		}
	}
	limiter.now = func() time.Time { return time.Date(2026, 8, 22, 3, 0, 0, 0, time.UTC) }
	if _, err := limiter.Allow(context.Background(), "203.0.113.8"); err != ErrLimited {
		t.Fatalf("error = %v, want ErrLimited", err)
	}
}

func TestLimiterStoresOnlyDerivedIdentifier(t *testing.T) {
	limiter, store := testLimiter(time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC))
	_, _ = limiter.Allow(context.Background(), "203.0.113.8")
	for key := range store.counts {
		if key == "203.0.113.8" || contains(key, "203.0.113.8") {
			t.Fatalf("raw IP persisted in key %q", key)
		}
	}
}
func contains(value string, part string) bool {
	return len(part) > 0 && len(value) >= len(part) && (value == part || containsAt(value, part))
}
func containsAt(value string, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
