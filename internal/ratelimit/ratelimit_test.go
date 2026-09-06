package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

type acceptedUpload struct {
	id   string
	size int64
	at   time.Time
}

type memoryStore struct {
	mu      sync.Mutex
	entries map[string][]acceptedUpload
}

func (s *memoryStore) Consume(_ context.Context, key string, id string, size int64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := now.Add(-RollingDay)
	kept := make([]acceptedUpload, 0, len(s.entries[key]))
	hourly := 0
	var total int64
	for _, entry := range s.entries[key] {
		if entry.at.After(cutoff) {
			if entry.id == id {
				return nil
			}
			kept = append(kept, entry)
			total += entry.size
			if entry.at.After(now.Add(-RollingHour)) {
				hourly++
			}
		}
	}
	if hourly >= MaxAcceptedJobsPerHour {
		return ErrLimited
	}
	if len(kept) >= MaxAcceptedJobsPerDay {
		return ErrDailyLimited
	}
	if total+size > MaxAcceptedBytesPerDay {
		return ErrByteLimited
	}
	s.entries[key] = append(kept, acceptedUpload{id: id, size: size, at: now})
	return nil
}

func testLimiter(now time.Time) (*Limiter, *memoryStore) {
	store := &memoryStore{entries: map[string][]acceptedUpload{}}
	limiter := New(store, "test-secret")
	limiter.now = func() time.Time { return now }
	return limiter, store
}

func allow(limiter *Limiter, id string, size int64) error {
	_, err := limiter.Allow(context.Background(), limiter.Identifier("203.0.113.8"), id, size)
	return err
}

func TestRollingLimitsAllowConfiguredCapacityAndRejectNext(t *testing.T) {
	limiter, _ := testLimiter(time.Date(2026, 8, 22, 17, 44, 0, 0, time.UTC))
	limit := min(MaxAcceptedJobsPerHour, MaxAcceptedJobsPerDay)
	for index := range limit {
		if err := allow(limiter, string(rune('a'+index)), 1); err != nil {
			t.Fatal(err)
		}
	}
	want := ErrLimited
	if MaxAcceptedJobsPerDay < MaxAcceptedJobsPerHour {
		want = ErrDailyLimited
	}
	if err := allow(limiter, "next", 1); err != want {
		t.Fatalf("error = %v", err)
	}
}

func TestRollingDayAllowsConfiguredCapacityAndRejectsNext(t *testing.T) {
	limiter, _ := testLimiter(time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	for index := range MaxAcceptedJobsPerDay {
		hour := index / MaxAcceptedJobsPerHour
		limiter.now = func() time.Time { return time.Date(2026, 8, 22, hour, 0, 0, 0, time.UTC) }
		if err := allow(limiter, string(rune(index+1)), 1); err != nil {
			t.Fatal(err)
		}
	}
	limiter.now = func() time.Time { return time.Date(2026, 8, 22, 3, 0, 0, 0, time.UTC) }
	if err := allow(limiter, "next", 1); err != ErrDailyLimited {
		t.Fatalf("error = %v", err)
	}
}

func TestByteAllowanceAndConcurrentAttempts(t *testing.T) {
	limiter, _ := testLimiter(time.Now().UTC())
	if err := allow(limiter, "full", MaxAcceptedBytesPerDay); err != nil {
		t.Fatal(err)
	}
	if err := allow(limiter, "next", 1); err != ErrByteLimited {
		t.Fatalf("error = %v", err)
	}
	limiter, _ = testLimiter(time.Now().UTC())
	var accepted int
	var lock sync.Mutex
	var group sync.WaitGroup
	for index := range 50 {
		group.Go(func() {
			if allow(limiter, string(rune(index+1)), 1) == nil {
				lock.Lock()
				accepted++
				lock.Unlock()
			}
		})
	}
	group.Wait()
	if accepted != min(MaxAcceptedJobsPerHour, MaxAcceptedJobsPerDay) {
		t.Fatalf("accepted = %d", accepted)
	}
}
