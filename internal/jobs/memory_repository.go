package jobs

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{jobs: make(map[string]Job)}
}

func (r *MemoryRepository) Create(_ context.Context, job Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.jobs[job.ID] = job
	return nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id string) (Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	return job, nil
}
