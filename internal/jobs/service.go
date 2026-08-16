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
	ErrFileTooLarge     = errors.New("fileSizeBytes must not exceed 26214400 bytes (25 MiB)")
)

const MaxUploadSizeBytes int64 = 25 * 1024 * 1024

type Service struct {
	repository     Repository
	uploadSigner   UploadSigner
	uploadLifetime time.Duration
}

type UploadSigner interface {
	PresignPut(ctx context.Context, objectKey string, fileSizeBytes int64, lifetime time.Duration) (url string, method string, err error)
}

func NewService(repository Repository, uploadSigner UploadSigner, uploadLifetime time.Duration) *Service {
	return &Service{
		repository:     repository,
		uploadSigner:   uploadSigner,
		uploadLifetime: uploadLifetime,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResult, error) {
	if strings.TrimSpace(input.FileName) == "" {
		return CreateResult{}, ErrFileNameRequired
	}
	if !strings.HasSuffix(strings.ToLower(input.FileName), ".stl") {
		return CreateResult{}, ErrFileNameNotSTL
	}
	if input.FileSizeBytes <= 0 {
		return CreateResult{}, ErrFileSizeInvalid
	}
	if input.FileSizeBytes > MaxUploadSizeBytes {
		return CreateResult{}, ErrFileTooLarge
	}

	id, err := newID()
	if err != nil {
		return CreateResult{}, err
	}

	createdAt := time.Now().UTC()
	job := Job{
		ID:            id,
		FileName:      input.FileName,
		FileSizeBytes: input.FileSizeBytes,
		Status:        StatusCreated,
		CreatedAt:     createdAt,
	}

	url, method, err := s.uploadSigner.PresignPut(ctx, "jobs/"+id+"/input.stl", input.FileSizeBytes, s.uploadLifetime)
	if err != nil {
		return CreateResult{}, err
	}
	if err := s.repository.Create(ctx, job); err != nil {
		return CreateResult{}, err
	}

	return CreateResult{
		Job: job,
		Upload: Upload{
			URL:       url,
			Method:    method,
			ExpiresAt: createdAt.Add(s.uploadLifetime),
		},
	}, nil
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
