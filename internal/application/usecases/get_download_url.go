package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/google/uuid"
)

const downloadURLTTL = 15 * time.Minute

// Presigner is the subset of storage.S3Client used by this use case,
// declared locally so it can be faked in unit tests.
type Presigner interface {
	PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type GetDownloadURLUseCase struct {
	videos  repositories.VideoRepository
	storage Presigner
}

func NewGetDownloadURLUseCase(videos repositories.VideoRepository, storage Presigner) *GetDownloadURLUseCase {
	return &GetDownloadURLUseCase{videos: videos, storage: storage}
}

// GetDownloadURL returns a short-lived presigned URL for a video's zip, once
// it has finished processing. It enforces ownership (ErrForbidden) and
// readiness (ErrNotReady).
func (uc *GetDownloadURLUseCase) GetDownloadURL(ctx context.Context, userID, videoID uuid.UUID) (string, error) {
	video, err := uc.videos.FindByID(ctx, videoID)
	if err != nil {
		return "", err
	}

	if video.UserID != userID {
		return "", repositories.ErrForbidden
	}
	if video.Status != entities.StatusCompleted || video.ZipObjectKey == nil {
		return "", repositories.ErrNotReady
	}

	url, err := uc.storage.PresignGetObject(ctx, *video.ZipObjectKey, downloadURLTTL)
	if err != nil {
		return "", fmt.Errorf("presign download url: %w", err)
	}
	return url, nil
}
