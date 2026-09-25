package entities

import (
	"time"

	"github.com/google/uuid"
)

type VideoStatus string

const (
	StatusUploaded   VideoStatus = "UPLOADED"
	StatusProcessing VideoStatus = "PROCESSING"
	StatusCompleted  VideoStatus = "COMPLETED"
	StatusFailed     VideoStatus = "FAILED"
)

// Video represents one uploaded source video and the lifecycle of its
// asynchronous frame-extraction processing, up to the downloadable zip.
type Video struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
	SourceBucket     string
	SourceObjectKey  string
	Status           VideoStatus
	ZipBucket        *string
	ZipObjectKey     *string
	FrameCount       *int
	ErrorMessage     *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
	FailedAt         *time.Time
}

func IsValidVideoStatus(s string) bool {
	switch VideoStatus(s) {
	case StatusUploaded, StatusProcessing, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}
