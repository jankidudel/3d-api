package jobs

import (
	"context"
	"errors"
	"testing"
)

func TestCreateValidJob(t *testing.T) {
	service := NewService(NewMemoryRepository())

	job, err := service.Create(context.Background(), CreateInput{FileName: "rotor.stl", FileSizeBytes: 117884})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if job.ID == "" {
		t.Error("Create() returned an empty ID")
	}
	if job.Status != StatusCreated {
		t.Errorf("Status = %q, want %q", job.Status, StatusCreated)
	}
	if job.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(NewMemoryRepository())
			_, err := service.Create(context.Background(), test.input)
			if !errors.Is(err, test.want) {
				t.Errorf("Create() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	service := NewService(NewMemoryRepository())
	created, err := service.Create(context.Background(), CreateInput{FileName: "rotor.stl", FileSizeBytes: 12})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got != created {
		t.Errorf("GetByID() = %+v, want %+v", got, created)
	}
}

func TestGetByIDMissing(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.GetByID(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, ErrNotFound)
	}
}
