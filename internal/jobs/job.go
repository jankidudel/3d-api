package jobs

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusCreated    Status = "created"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Job struct {
	ID            string
	FileName      string
	FileSizeBytes int64
	Status        Status
	CreatedAt     time.Time
	Report        json.RawMessage
	Failure       json.RawMessage
}

type CreateInput struct {
	FileName      string
	FileSizeBytes int64
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
