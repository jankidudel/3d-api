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
	InputFolder  = "input"
	OutputFolder = "output"
)

const (
	OperationQualityCheck      Operation = "quality_check"
	OperationSafeStlRepair     Operation = "safe_stl_repair"
	OperationSafeThreeMfRepair Operation = "safe_3mf_repair"
	OperationStlSplit          Operation = "stl_split"
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
	if operation == OperationSafeStlRepair && f == FileFormatSTL {
		return InputFolder + "/repair.stl", nil
	}
	if operation == OperationStlSplit && f == FileFormatSTL {
		return InputFolder + "/split.stl", nil
	}
	if operation == OperationSafeThreeMfRepair && f == FileFormat3MF {
		return InputFolder + "/repair.3mf", nil
	}
	definition, ok := f.Definition()
	if !ok {
		return "", errors.New("unsupported file format")
	}
	return InputFolder + "/input" + definition.Extension, nil
}

const (
	StatusCreated    Status = "created"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Job struct {
	ID                   string
	FileName             string
	FileFormat           FileFormat
	Operation            Operation
	SplitAxis            string
	SplitPositionPercent float64
	FileSizeBytes        int64
	QuotaID              string
	Status               Status
	CreatedAt            time.Time
	Report               json.RawMessage
	Failure              json.RawMessage
}

type CreateInput struct {
	FileName             string
	FileSizeBytes        int64
	FileFormat           FileFormat
	Operation            Operation
	SplitAxis            string
	SplitPositionPercent float64
	QuotaID              string
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
