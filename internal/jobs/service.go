package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrFileNameRequired = errors.New("fileName is required")
	ErrFileNameNotSTL   = errors.New("fileName must end in .stl")
	ErrFileSizeInvalid  = errors.New("fileSizeBytes must be greater than 0")
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Job, error) {
	if strings.TrimSpace(input.FileName) == "" {
		return Job{}, ErrFileNameRequired
	}
	if !strings.HasSuffix(strings.ToLower(input.FileName), ".stl") {
		return Job{}, ErrFileNameNotSTL
	}
	if input.FileSizeBytes <= 0 {
		return Job{}, ErrFileSizeInvalid
	}

	id, err := newID()
	if err != nil {
		return Job{}, err
	}

	job := Job{
		ID:            id,
		FileName:      input.FileName,
		FileSizeBytes: input.FileSizeBytes,
		Status:        StatusCreated,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Job, error) {
	return s.repository.GetByID(ctx, id)
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
