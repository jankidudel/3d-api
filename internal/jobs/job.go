package jobs

import (
	"encoding/json"
	"errors"
	"time"
)

type Status string
type FileFormat string

const (
	FileFormatSTL FileFormat = "stl"
	FileFormat3MF FileFormat = "3mf"
)

type FormatDefinition struct {
	Extension      string
	InvalidNameErr error
}

var formatDefinitions = map[FileFormat]FormatDefinition{
	FileFormatSTL: {Extension: ".stl", InvalidNameErr: ErrFileNameNotSTL},
	FileFormat3MF: {Extension: ".3mf", InvalidNameErr: ErrFileNameNot3MF},
}

func (f FileFormat) Definition() (FormatDefinition, bool) {
	definition, ok := formatDefinitions[f]
	return definition, ok
}

func (f FileFormat) InputObjectName() (string, error) {
	definition, ok := f.Definition()
	if !ok {
		return "", errors.New("unsupported file format")
	}
	return "input" + definition.Extension, nil
}

const (
	StatusCreated    Status = "created"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Job struct {
	ID            string
	FileName      string
	FileFormat    FileFormat
	FileSizeBytes int64
	Status        Status
	CreatedAt     time.Time
	Report        json.RawMessage
	Failure       json.RawMessage
}

type CreateInput struct {
	FileName      string
	FileSizeBytes int64
	FileFormat    FileFormat
}

type Upload struct {
	URL       string
	Method    string
	Fields    map[string]string
	ExpiresAt time.Time
}

type CreateResult struct {
	Job    Job
	Upload Upload
}
