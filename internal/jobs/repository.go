package jobs

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("job not found")

type Repository interface {
	Create(ctx context.Context, job Job) error
	GetByID(ctx context.Context, id string) (Job, error)
	InputSize(ctx context.Context, job Job) (int64, error)
}
