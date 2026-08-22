package jobs

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

const testUploadLifetime = 15 * time.Minute

type stubUploadSigner struct {
	url       string
	fields    map[string]string
	objectKey string
	fileSize  int64
	lifetime  time.Duration
	err       error
}

func (s *stubUploadSigner) PresignPost(
	_ context.Context,
	objectKey string,
	fileSizeBytes int64,
	lifetime time.Duration,
) (string, map[string]string, error) {
	s.objectKey = objectKey
	s.fileSize = fileSizeBytes
	s.lifetime = lifetime
	return s.url, s.fields, s.err
}

func newTestService() (*Service, *stubUploadSigner) {
	signer := &stubUploadSigner{
		url:    "https://uploads.example/presigned",
		fields: map[string]string{"policy": "signed-policy"},
	}
	return NewService(NewMemoryRepository(), signer, testUploadLifetime), signer
}

func TestCreateValidJob(t *testing.T) {
	service, signer := newTestService()

	result, err := service.Create(context.Background(), CreateInput{FileName: "rotor.stl", FileSizeBytes: 117884})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	job := result.Job
	if job.ID == "" {
		t.Error("Create() returned an empty ID")
	}
	if job.Status != StatusCreated {
		t.Errorf("Status = %q, want %q", job.Status, StatusCreated)
	}
	if job.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if signer.objectKey != "jobs/"+job.ID+"/input.stl" {
		t.Errorf("object key = %q", signer.objectKey)
	}
	if signer.fileSize != job.FileSizeBytes {
		t.Errorf("signed file size = %d, want %d", signer.fileSize, job.FileSizeBytes)
	}
	if signer.lifetime != testUploadLifetime {
		t.Errorf("signed lifetime = %v, want %v", signer.lifetime, testUploadLifetime)
	}
	if result.Upload.URL != signer.url || result.Upload.Method != "POST" || !reflect.DeepEqual(result.Upload.Fields, signer.fields) {
		t.Errorf("Upload = %+v", result.Upload)
	}
	if want := job.CreatedAt.Add(testUploadLifetime); !result.Upload.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", result.Upload.ExpiresAt, want)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input CreateInput
		want  error
	}{
		{name: "missing filename", input: CreateInput{FileSizeBytes: 1}, want: ErrFileNameRequired},
		{name: "non STL filename", input: CreateInput{FileName: "rotor.obj", FileSizeBytes: 1}, want: ErrFileNameNotSTL},
		{name: "zero size", input: CreateInput{FileName: "rotor.stl"}, want: ErrFileSizeInvalid},
		{name: "negative size", input: CreateInput{FileName: "rotor.stl", FileSizeBytes: -1}, want: ErrFileSizeInvalid},
		{name: "too large", input: CreateInput{FileName: "rotor.stl", FileSizeBytes: MaxUploadSizeBytes + 1}, want: ErrFileTooLarge},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, _ := newTestService()
			_, err := service.Create(context.Background(), test.input)
			if !errors.Is(err, test.want) {
				t.Errorf("Create() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestCreateAcceptsExactlyMaximumUploadSize(t *testing.T) {
	service, _ := newTestService()
	if _, err := service.Create(context.Background(), CreateInput{FileName: "rotor.stl", FileSizeBytes: MaxUploadSizeBytes}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestGetByID(t *testing.T) {
	service, _ := newTestService()
	result, err := service.Create(context.Background(), CreateInput{FileName: "rotor.stl", FileSizeBytes: 12})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := service.GetByID(context.Background(), result.Job.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if !reflect.DeepEqual(got, result.Job) {
		t.Errorf("GetByID() = %+v, want %+v", got, result.Job)
	}
}

func TestGetByIDMissing(t *testing.T) {
	service, _ := newTestService()
	_, err := service.GetByID(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, ErrNotFound)
	}
}

func TestCreateDoesNotPersistJobWhenSigningFails(t *testing.T) {
	repository := NewMemoryRepository()
	signer := &stubUploadSigner{err: errors.New("signing failed")}
	service := NewService(repository, signer, testUploadLifetime)

	_, err := service.Create(context.Background(), CreateInput{FileName: "rotor.stl", FileSizeBytes: 1})

	if err == nil {
		t.Fatal("Create() error = nil, want signing error")
	}
	if len(repository.jobs) != 0 {
		t.Errorf("repository contains %d jobs after signing failure, want 0", len(repository.jobs))
	}
}
