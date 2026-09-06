package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrFileNameRequired = errors.New("fileName is required")
	ErrFileNameNotSTL   = errors.New("fileName must end in .stl")
	ErrFileNameNot3MF   = errors.New("fileName must end in .3mf")
	ErrFileSizeInvalid  = errors.New("fileSizeBytes must be greater than 0")
	ErrFileTooLarge     = fmt.Errorf("fileSizeBytes must not exceed %d bytes (%d MiB)", MaxUploadSizeBytes, MaxUploadSizeMiB)
)

const (
	MaxUploadSizeMiB    int64 = 25
	MaxUploadSizeBytes        = MaxUploadSizeMiB * 1024 * 1024
	FileTooLargeMessage       = "Files must be 25 MB or smaller."
)

type Service struct {
	repository     Repository
	dispatcher     UploadDispatcher
	uploadSigner   UploadSigner
	uploadLifetime time.Duration
}

type UploadDispatcher interface {
	Dispatch(context.Context, Job, int64) error
}

type UploadSigner interface {
	PresignPost(ctx context.Context, objectKey string, fileSizeBytes int64, lifetime time.Duration) (url string, fields map[string]string, err error)
}

func NewService(repository Repository, uploadSigner UploadSigner, uploadLifetime time.Duration, dispatchers ...UploadDispatcher) *Service {
	var dispatcher UploadDispatcher
	if len(dispatchers) > 0 {
		dispatcher = dispatchers[0]
	}
	return &Service{
		repository:     repository,
		dispatcher:     dispatcher,
		uploadSigner:   uploadSigner,
		uploadLifetime: uploadLifetime,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResult, error) {
	if input.FileFormat == "" {
		input.FileFormat = FileFormatSTL
	}
	if input.Operation == "" {
		input.Operation = OperationQualityCheck
	}
	if err := ValidateCreateInput(input); err != nil {
		return CreateResult{}, err
	}
	id, err := newID()
	if err != nil {
		return CreateResult{}, err
	}

	createdAt := time.Now().UTC()
	job := Job{
		ID:            id,
		FileName:      input.FileName,
		FileSizeBytes: input.FileSizeBytes, QuotaID: input.QuotaID,
		FileFormat: input.FileFormat,
		Operation:  input.Operation, SplitAxis: input.SplitAxis, SplitPositionPercent: input.SplitPositionPercent,
		Status:    StatusCreated,
		CreatedAt: createdAt,
	}

	inputName, err := input.FileFormat.InputObjectName(input.Operation)
	if err != nil {
		return CreateResult{}, err
	}
	url, fields, err := s.uploadSigner.PresignPost(ctx, "jobs/"+id+"/"+inputName, input.FileSizeBytes, s.uploadLifetime)
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
			Method:    "POST",
			Fields:    fields,
			ExpiresAt: createdAt.Add(s.uploadLifetime),
		},
	}, nil
}

func (s *Service) AcceptUpload(ctx context.Context, id string) (Job, int64, error) {
	job, err := s.GetByID(ctx, id)
	if err != nil {
		return Job{}, 0, err
	}
	size, err := s.repository.InputSize(ctx, job)
	if err != nil {
		return Job{}, 0, err
	}
	if size <= 0 || size > MaxUploadSizeBytes {
		return Job{}, size, ErrFileTooLarge
	}
	return job, size, nil
}

func (s *Service) QueueUpload(ctx context.Context, job Job, actualSize int64) error {
	if s.dispatcher == nil {
		return errors.New("upload dispatcher is not configured")
	}
	return s.dispatcher.Dispatch(ctx, job, actualSize)
}

func ValidateCreateInput(input CreateInput) error {
	if strings.TrimSpace(input.FileName) == "" {
		return ErrFileNameRequired
	}
	format := input.FileFormat
	if format == "" {
		format = FileFormatSTL
	}
	definition, ok := format.Definition()
	if !ok {
		return errors.New("unsupported file format")
	}
	if !strings.HasSuffix(strings.ToLower(input.FileName), definition.Extension) {
		return definition.InvalidNameErr
	}
	if input.Operation != "" && input.Operation != OperationQualityCheck && input.Operation != OperationSafeStlRepair && input.Operation != OperationSafeThreeMfRepair && input.Operation != OperationStlSplit {
		return errors.New("unsupported job operation")
	}
	if input.Operation == OperationSafeStlRepair && format != FileFormatSTL {
		return errors.New("safe STL repair accepts STL files only")
	}
	if input.Operation == OperationSafeThreeMfRepair && format != FileFormat3MF {
		return errors.New("safe 3MF repair accepts 3MF files only")
	}
	if input.Operation == OperationStlSplit && (format != FileFormatSTL || (input.SplitAxis != "X" && input.SplitAxis != "Y" && input.SplitAxis != "Z") || input.SplitPositionPercent <= 0 || input.SplitPositionPercent >= 100) {
		return errors.New("STL split requires an STL, X/Y/Z axis and a position inside the model bounds")
	}
	if input.FileSizeBytes <= 0 {
		return ErrFileSizeInvalid
	}
	if input.FileSizeBytes > MaxUploadSizeBytes {
		return ErrFileTooLarge
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Job, error) {
	if len(id) != 32 {
		return Job{}, ErrNotFound
	}
	if _, err := hex.DecodeString(id); err != nil {
		return Job{}, ErrNotFound
	}
	return s.repository.GetByID(ctx, id)
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
