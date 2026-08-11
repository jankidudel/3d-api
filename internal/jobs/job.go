package jobs

import "time"

type Status string

const StatusCreated Status = "created"

type Job struct {
	ID            string
	FileName      string
	FileSizeBytes int64
	Status        Status
	CreatedAt     time.Time
}

type CreateInput struct {
	FileName      string
	FileSizeBytes int64
}
