package usecases

import (
	"context"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
)

func newStatusFailedCommand(t *testing.T, videoID uuid.UUID) messaging.Event {
	t.Helper()
	ev, err := messaging.NewEvent("video.status.failed", "corr-x", "", videoStatusFailedPayload{
		VideoID:      videoID.String(),
		ErrorCode:    "FRAME_EXTRACTION_FAILED",
		ErrorMessage: "codec not supported",
		FailedAt:     "2026-09-25T10:00:00.000Z",
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}
	return ev
}

func TestHandleStatusFailed_Success(t *testing.T) {
	videos := newFakeVideoRepository()
	processed := newFakeProcessedEventRepository()
	uc := NewHandleStatusFailedUseCase(videos, processed)

	video := entities.Video{ID: uuid.New(), UserID: uuid.New(), Status: entities.StatusProcessing}
	if err := videos.Save(context.Background(), &video, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	ev := newStatusFailedCommand(t, video.ID)
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	saved, err := videos.FindByID(context.Background(), video.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if saved.Status != entities.StatusFailed {
		t.Errorf("Status = %q, want %q", saved.Status, entities.StatusFailed)
	}
	if saved.ErrorMessage == nil || *saved.ErrorMessage != "codec not supported" {
		t.Errorf("ErrorMessage = %v, want %q", saved.ErrorMessage, "codec not supported")
	}
	if saved.FailedAt == nil {
		t.Error("FailedAt should be set")
	}

	eventID, _ := uuid.Parse(ev.EventID)
	isProcessed, err := processed.IsProcessed(context.Background(), eventID)
	if err != nil {
		t.Fatalf("IsProcessed() error = %v", err)
	}
	if !isProcessed {
		t.Error("expected event to be marked processed")
	}
}

func TestHandleStatusFailed_Idempotent(t *testing.T) {
	videos := newFakeVideoRepository()
	processed := newFakeProcessedEventRepository()
	uc := NewHandleStatusFailedUseCase(videos, processed)

	video := entities.Video{ID: uuid.New(), UserID: uuid.New(), Status: entities.StatusProcessing}
	if err := videos.Save(context.Background(), &video, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	ev := newStatusFailedCommand(t, video.ID)
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("first Handle() error = %v", err)
	}
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("second Handle() error = %v", err)
	}
}

func TestHandleStatusFailed_VideoNotFound(t *testing.T) {
	videos := newFakeVideoRepository()
	processed := newFakeProcessedEventRepository()
	uc := NewHandleStatusFailedUseCase(videos, processed)

	ev := newStatusFailedCommand(t, uuid.New())
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Errorf("Handle() error = %v, want nil for unknown video", err)
	}
}
