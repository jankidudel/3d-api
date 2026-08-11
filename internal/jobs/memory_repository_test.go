package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryRepositoryCreatesAndRetrievesJob(t *testing.T) {
	repository := NewMemoryRepository()
	want := Job{
		ID:            "job-1",
		FileName:      "rotor.stl",
		FileSizeBytes: 42,
		Status:        StatusCreated,
		CreatedAt:     time.Now().UTC(),
	}

	if err := repository.Create(context.Background(), want); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repository.GetByID(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got != want {
		t.Errorf("GetByID() = %+v, want %+v", got, want)
	}
}

func TestMemoryRepositoryReturnsNotFound(t *testing.T) {
	repository := NewMemoryRepository()

	_, err := repository.GetByID(context.Background(), "missing")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryRepositorySupportsConcurrentAccess(t *testing.T) {
	repository := NewMemoryRepository()
	const jobCount = 100
	var waitGroup sync.WaitGroup

	for index := range jobCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			id := fmt.Sprintf("job-%d", index)
			job := Job{ID: id, FileName: "rotor.stl", FileSizeBytes: 1, Status: StatusCreated}
			if err := repository.Create(context.Background(), job); err != nil {
				t.Errorf("Create(%q) error = %v", id, err)
				return
			}
			if _, err := repository.GetByID(context.Background(), id); err != nil {
				t.Errorf("GetByID(%q) error = %v", id, err)
			}
		}()
	}

	waitGroup.Wait()
}
