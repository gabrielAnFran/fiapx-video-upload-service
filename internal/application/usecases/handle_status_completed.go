package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
)

type HandleStatusCompletedUseCase struct {
	videos          repositories.VideoRepository
	processedEvents repositories.ProcessedEventRepository
}

func NewHandleStatusCompletedUseCase(videos repositories.VideoRepository, processedEvents repositories.ProcessedEventRepository) *HandleStatusCompletedUseCase {
	return &HandleStatusCompletedUseCase{videos: videos, processedEvents: processedEvents}
}

type videoStatusCompletedPayload struct {
	VideoID      string `json:"video_id"`
	ZipBucket    string `json:"zip_bucket"`
	ZipObjectKey string `json:"zip_object_key"`
	FrameCount   int    `json:"frame_count"`
	CompletedAt  string `json:"completed_at"`
}

// Handle processes a video.status.completed command emitted by the saga
// orchestrator, idempotently. This service is a terminal consumer of the
// event: it does not emit any further event in response.
func (uc *HandleStatusCompletedUseCase) Handle(ctx context.Context, ev messaging.Event) error {
	eventID, err := uuid.Parse(ev.EventID)
	if err != nil {
		return fmt.Errorf("invalid event id: %w", err)
	}

	processed, err := uc.processedEvents.IsProcessed(ctx, eventID)
	if err != nil {
		return fmt.Errorf("check processed: %w", err)
	}
	if processed {
		return nil
	}

	var cmd videoStatusCompletedPayload
	if err := json.Unmarshal(ev.Payload, &cmd); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	videoID, err := uuid.Parse(cmd.VideoID)
	if err != nil {
		return fmt.Errorf("invalid video_id: %w", err)
	}

	video, err := uc.videos.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil // nothing to update, treat as processed
		}
		return fmt.Errorf("find video: %w", err)
	}

	completedAt, err := time.Parse("2006-01-02T15:04:05.000Z07:00", cmd.CompletedAt)
	if err != nil {
		completedAt = time.Now().UTC()
	}

	video.Status = entities.StatusCompleted
	video.ZipBucket = &cmd.ZipBucket
	video.ZipObjectKey = &cmd.ZipObjectKey
	video.FrameCount = &cmd.FrameCount
	video.CompletedAt = &completedAt
	video.UpdatedAt = time.Now().UTC()

	if err := uc.videos.Save(ctx, video, nil); err != nil {
		return fmt.Errorf("save video: %w", err)
	}

	if err := uc.processedEvents.MarkProcessed(ctx, eventID); err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return nil
}
