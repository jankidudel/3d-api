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

func (s *contactMemoryStore) Decrement(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts[key] > 0 {
		s.counts[key]--
	}
	return nil
}

func TestContactLimiterEnforcesPerIPDailyLimit(t *testing.T) {
	limiter := NewContactLimiter(&contactMemoryStore{counts: map[string]int{}}, "test-secret")
	limiter.now = func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
	for range MaxContactMessagesPerIPDay {
		if _, err := limiter.Reserve(context.Background(), "203.0.113.8:1234"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := limiter.Reserve(context.Background(), "203.0.113.8:4321"); err != ErrContactDailyLimited {
		t.Fatalf("error = %v, want %v", err, ErrContactDailyLimited)
	}
}

func TestContactLimiterEnforcesGlobalDailyLimit(t *testing.T) {
	limiter := NewContactLimiter(&contactMemoryStore{counts: map[string]int{}}, "test-secret")
	limiter.now = func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
	for index := range MaxContactMessagesPerDay {
		if _, err := limiter.Reserve(context.Background(), "203.0.113."+strconv.Itoa(index)+":1234"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := limiter.Reserve(context.Background(), "198.51.100.8:1234"); err != ErrContactGlobalLimited {
		t.Fatalf("error = %v, want %v", err, ErrContactGlobalLimited)
	}
}

func TestContactLimiterReleasesAFailedDeliveryReservation(t *testing.T) {
	store := &contactMemoryStore{counts: map[string]int{}}
	limiter := NewContactLimiter(store, "test-secret")
	limiter.now = func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
	reservation, err := limiter.Reserve(context.Background(), "203.0.113.8:1234")
	if err != nil {
		t.Fatal(err)
	}
	if err := limiter.Release(context.Background(), reservation); err != nil {
		t.Fatal(err)
	}
	for range MaxContactMessagesPerIPDay {
		if _, err := limiter.Reserve(context.Background(), "203.0.113.8:1234"); err != nil {
			t.Fatal(err)
		}
	}
}
