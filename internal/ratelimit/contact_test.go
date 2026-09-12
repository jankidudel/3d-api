package ratelimit

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

type contactMemoryStore struct {
	mu     sync.Mutex
	counts map[string]int
}

func (s *contactMemoryStore) Increment(_ context.Context, key string, limit int, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts[key] >= limit {
		return ErrLimited
	}
	s.counts[key]++
	return nil
}

func TestContactLimiterEnforcesPerIPHourlyLimit(t *testing.T) {
	limiter := NewContactLimiter(&contactMemoryStore{counts: map[string]int{}}, "test-secret")
	limiter.now = func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
	for range MaxContactMessagesPerIPHour {
		if err := limiter.Allow(context.Background(), "203.0.113.8:1234"); err != nil {
			t.Fatal(err)
		}
	}
	if err := limiter.Allow(context.Background(), "203.0.113.8:4321"); err != ErrContactHourlyLimited {
		t.Fatalf("error = %v, want %v", err, ErrContactHourlyLimited)
	}
}

func TestContactLimiterEnforcesGlobalDailyLimit(t *testing.T) {
	limiter := NewContactLimiter(&contactMemoryStore{counts: map[string]int{}}, "test-secret")
	limiter.now = func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
	for index := range MaxContactMessagesPerDay {
		if err := limiter.Allow(context.Background(), "203.0.113."+strconv.Itoa(index)+":1234"); err != nil {
			t.Fatal(err)
		}
	}
	if err := limiter.Allow(context.Background(), "198.51.100.8:1234"); err != ErrContactGlobalLimited {
		t.Fatalf("error = %v, want %v", err, ErrContactGlobalLimited)
	}
}
