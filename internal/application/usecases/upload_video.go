package usecases

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
)

const eventVideoUploaded = "video.uploaded"

// Uploader is the subset of storage.S3Client used by this use case,
// declared locally so it can be faked in unit tests.
type Uploader interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error
}

type UploadVideoUseCase struct {
	videos       repositories.VideoRepository
	users        repositories.UserRepository
	storage      Uploader
	sourceBucket string
}

func NewUploadVideoUseCase(videos repositories.VideoRepository, users repositories.UserRepository, storage Uploader, sourceBucket string) *UploadVideoUseCase {
	return &UploadVideoUseCase{videos: videos, users: users, storage: storage, sourceBucket: sourceBucket}
}

type videoUploadedPayload struct {
	VideoID          string `json:"video_id"`
	UserID           string `json:"user_id"`
	UserEmail        string `json:"user_email"`
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type"`
	SizeBytes        int64  `json:"size_bytes"`
	SourceBucket     string `json:"source_bucket"`
	SourceObjectKey  string `json:"source_object_key"`
	UploadedAt       string `json:"uploaded_at"`
}

// UploadVideo streams the given body into object storage, records a new
// video row (status UPLOADED), and enqueues a video.uploaded outbox event,
// all atomically with the domain write.
func (uc *UploadVideoUseCase) UploadVideo(ctx context.Context, userID uuid.UUID, filename, contentType string, size int64, body io.Reader) (*entities.Video, error) {
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	videoID := uuid.New()
	objectKey := fmt.Sprintf("raw/%s/%s/original%s", userID.String(), videoID.String(), path.Ext(filename))

	if err := uc.storage.Upload(ctx, objectKey, body, contentType); err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	now := time.Now().UTC()
	video := &entities.Video{
		ID:               videoID,
		UserID:           userID,
		OriginalFilename: filename,
		ContentType:      contentType,
		SizeBytes:        size,
		SourceBucket:     uc.sourceBucket,
		SourceObjectKey:  objectKey,
		Status:           entities.StatusUploaded,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	payload := videoUploadedPayload{
		VideoID:          video.ID.String(),
		UserID:           video.UserID.String(),
		UserEmail:        user.Email,
		OriginalFilename: video.OriginalFilename,
		ContentType:      video.ContentType,
		SizeBytes:        video.SizeBytes,
		SourceBucket:     video.SourceBucket,
		SourceObjectKey:  video.SourceObjectKey,
		UploadedAt:       now.Format("2006-01-02T15:04:05.000Z07:00"),
	}

	env, err := messaging.NewEvent(eventVideoUploaded, videoID.String(), "", payload)
	if err != nil {
		return nil, fmt.Errorf("build event: %w", err)
	}
	envelope, err := marshalEnvelope(env)
	if err != nil {
		return nil, err
	}

	outboxEvent := &repositories.OutboxEvent{
		EventID:     uuid.MustParse(env.EventID),
		AggregateID: video.ID,
		EventName:   env.EventName,
		Payload:     envelope,
		Headers:     []byte(`{"content-type":"application/json"}`),
	}

	if err := uc.videos.Save(ctx, video, outboxEvent); err != nil {
		return nil, fmt.Errorf("save video: %w", err)
	}

	return video, nil
}
