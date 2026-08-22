package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu      sync.Mutex
	entries map[string][]time.Time
}

func (s *memoryStore) Consume(_ context.Context, id string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := now.Add(-RollingDay)
	kept := []time.Time{}
	hourly := 0
	for _, t := range s.entries[id] {
		if t.After(cutoff) {
			kept = append(kept, t)
			if t.After(now.Add(-RollingHour)) {
				hourly++
			}
		}
	}
	if hourly >= MaxAcceptedJobsPerHour || len(kept) >= MaxAcceptedJobsPerDay {
		return ErrLimited
	}
	s.entries[id] = append(kept, now)
	return nil
}
func testLimiter(now time.Time) (*Limiter, *memoryStore) {
	store := &memoryStore{entries: map[string][]time.Time{}}
	l := New(store, "test-secret")
	l.now = func() time.Time { return now }
	return l, store
}
func TestRollingHour(t *testing.T) {
	l, _ := testLimiter(time.Date(2026, 8, 22, 17, 44, 0, 0, time.UTC))
	for range 10 {
		if _, e := l.Allow(context.Background(), "203.0.113.8"); e != nil {
			t.Fatal(e)
		}
	}
	l.now = func() time.Time { return time.Date(2026, 8, 22, 18, 0, 0, 0, time.UTC) }
	if _, e := l.Allow(context.Background(), "203.0.113.8"); e != ErrLimited {
		t.Fatalf("boundary error = %v", e)
	}
	l.now = func() time.Time { return time.Date(2026, 8, 22, 18, 44, 0, 0, time.UTC) }
	if _, e := l.Allow(context.Background(), "203.0.113.8"); e != nil {
		t.Fatal(e)
	}
}
func TestRollingDay(t *testing.T) {
	l, _ := testLimiter(time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	for hour := 0; hour < 3; hour++ {
		l.now = func() time.Time { return time.Date(2026, 8, 22, hour, 0, 0, 0, time.UTC) }
		for range 10 {
			if _, e := l.Allow(context.Background(), "203.0.113.8"); e != nil {
				t.Fatal(e)
			}
		}
	}
	l.now = func() time.Time { return time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC) }
	if _, e := l.Allow(context.Background(), "203.0.113.8"); e != nil {
		t.Fatal(e)
	}
}
func TestConcurrentRequestsCannotExceedLimit(t *testing.T) {
	l, _ := testLimiter(time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC))
	var accepted int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if _, e := l.Allow(context.Background(), "203.0.113.8"); e == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if accepted != 10 {
		t.Fatalf("accepted=%d", accepted)
	}
}
func TestLimiterStoresOnlyDerivedIdentifier(t *testing.T) {
	l, s := testLimiter(time.Now())
	_, _ = l.Allow(context.Background(), "203.0.113.8")
	for k := range s.entries {
		if k == "203.0.113.8" {
			t.Fatal("raw IP stored")
		}
	}
}
