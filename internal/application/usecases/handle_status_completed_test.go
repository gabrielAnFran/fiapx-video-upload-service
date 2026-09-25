package usecases

import (
	"context"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
)

func newStatusCompletedCommand(t *testing.T, videoID uuid.UUID) messaging.Event {
	t.Helper()
	ev, err := messaging.NewEvent("video.status.completed", "corr-x", "", videoStatusCompletedPayload{
		VideoID:      videoID.String(),
		ZipBucket:    "fiapx-videos",
		ZipObjectKey: "zips/" + videoID.String() + "/frames.zip",
		FrameCount:   42,
		CompletedAt:  "2026-09-25T10:00:00.000Z",
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}
	return ev
}

func TestHandleStatusCompleted_Success(t *testing.T) {
	videos := newFakeVideoRepository()
	processed := newFakeProcessedEventRepository()
	uc := NewHandleStatusCompletedUseCase(videos, processed)

	video := entities.Video{ID: uuid.New(), UserID: uuid.New(), Status: entities.StatusProcessing}
	if err := videos.Save(context.Background(), &video, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	ev := newStatusCompletedCommand(t, video.ID)
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	saved, err := videos.FindByID(context.Background(), video.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if saved.Status != entities.StatusCompleted {
		t.Errorf("Status = %q, want %q", saved.Status, entities.StatusCompleted)
	}
	if saved.ZipObjectKey == nil || *saved.ZipObjectKey != "zips/"+video.ID.String()+"/frames.zip" {
		t.Errorf("ZipObjectKey = %v, want zips/%s/frames.zip", saved.ZipObjectKey, video.ID.String())
	}
	if saved.FrameCount == nil || *saved.FrameCount != 42 {
		t.Errorf("FrameCount = %v, want 42", saved.FrameCount)
	}
	if saved.CompletedAt == nil {
		t.Error("CompletedAt should be set")
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

func TestHandleStatusCompleted_Idempotent(t *testing.T) {
	videos := newFakeVideoRepository()
	processed := newFakeProcessedEventRepository()
	uc := NewHandleStatusCompletedUseCase(videos, processed)

	video := entities.Video{ID: uuid.New(), UserID: uuid.New(), Status: entities.StatusProcessing}
	if err := videos.Save(context.Background(), &video, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	ev := newStatusCompletedCommand(t, video.ID)
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("first Handle() error = %v", err)
	}
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("second Handle() error = %v", err)
	}

	saved, err := videos.FindByID(context.Background(), video.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if saved.Status != entities.StatusCompleted {
		t.Errorf("Status = %q, want %q", saved.Status, entities.StatusCompleted)
	}
}

func TestHandleStatusCompleted_VideoNotFound(t *testing.T) {
	videos := newFakeVideoRepository()
	processed := newFakeProcessedEventRepository()
	uc := NewHandleStatusCompletedUseCase(videos, processed)

	ev := newStatusCompletedCommand(t, uuid.New())
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Errorf("Handle() error = %v, want nil for unknown video", err)
	}
}
