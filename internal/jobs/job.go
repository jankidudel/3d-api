package jobs

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// DetectFileFormat derives the accepted format from the submitted filename.
// The API never accepts a client-selected format as authoritative.
func DetectFileFormat(fileName string) (FileFormat, error) {
	name := strings.ToLower(strings.TrimSpace(fileName))
	if name == "" {
		return "", ErrFileNameRequired
	}
	for format, definition := range formatDefinitions {
		if strings.HasSuffix(name, definition.Extension) {
			return format, nil
		}
	}
	return "", errors.New("fileName must end in .stl or .3mf")
}

type Status string
type FileFormat string
type Operation string

const (
	FileFormatSTL FileFormat = "stl"
	FileFormat3MF FileFormat = "3mf"
)

const (
	OperationQualityCheck          Operation = "quality_check"
	OperationThreeMfQualityChecker Operation = "3mf_quality_checker"
	OperationSafeStlRepair         Operation = "safe_stl_repair"
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

func (f FileFormat) InputObjectName(operation Operation) (string, error) {
	if operation == OperationThreeMfQualityChecker && f == FileFormat3MF {
		return "3mf_quality_checker.3mf", nil
	}
	if operation == OperationSafeStlRepair && f == FileFormatSTL {
		return "repair.stl", nil
	}
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
	Operation     Operation
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
	Operation     Operation
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
