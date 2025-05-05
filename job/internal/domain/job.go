package domain

import (
	"github.com/google/uuid"
	"time"
)

type JobStatus int

const (
	Pending JobStatus = iota
	Processing
	Converting
	Completed
	Failed
)

func (p JobStatus) String() string {
	return [...]string{"Pending", "Processing", "Converting", "Completed", "Failed"}[p]
}

func (p JobStatus) Index() int {
	return int(p)
}

type Job struct {
	ID      string
	Status  JobStatus
	FileKey string
	// File key in S3.
	// Will store the initial file key.
	// Serve the converted filekey only if status = Completed by giving it a prefix.
	//
	// Maybe we can re-encrypt the initial key with the prefix so that it gives different key.
	// Instead of "prefix/encrypted_key", it'll be just "encrypted_key"

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewJob(fileKey string) *Job {
	return &Job{
		ID:        uuid.NewString(),
		Status:    Pending,
		FileKey:   fileKey,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (j *Job) SetStatus(status JobStatus) {
	j.Status = status
	j.UpdatedAt = time.Now()
}
