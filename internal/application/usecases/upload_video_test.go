package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
)

func TestUploadVideo_Success(t *testing.T) {
	users := newFakeUserRepository()
	videos := newFakeVideoRepository()
	uploader := newFakeUploader()

	user := entities.User{ID: uuid.New(), Email: "uploader@example.com"}
	if err := users.Create(context.Background(), &user); err != nil {
		t.Fatalf("setup Create user error = %v", err)
	}

	uc := NewUploadVideoUseCase(videos, users, uploader, "fiapx-videos")

	body := strings.NewReader("fake video bytes")
	video, err := uc.UploadVideo(context.Background(), user.ID, "myvideo.mp4", "video/mp4", int64(body.Len()), body)
	if err != nil {
		t.Fatalf("UploadVideo() error = %v", err)
	}

	if video.Status != entities.StatusUploaded {
		t.Errorf("Status = %q, want %q", video.Status, entities.StatusUploaded)
	}
	if video.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", video.UserID, user.ID)
	}
	if video.SourceBucket != "fiapx-videos" {
		t.Errorf("SourceBucket = %q, want %q", video.SourceBucket, "fiapx-videos")
	}
	wantKey := "raw/" + user.ID.String() + "/" + video.ID.String() + "/original.mp4"
	if video.SourceObjectKey != wantKey {
		t.Errorf("SourceObjectKey = %q, want %q", video.SourceObjectKey, wantKey)
	}

	if _, ok := uploader.uploaded[wantKey]; !ok {
		t.Fatalf("expected object %q to have been uploaded", wantKey)
	}

	saved, err := videos.FindByID(context.Background(), video.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if saved.Status != entities.StatusUploaded {
		t.Errorf("saved Status = %q, want %q", saved.Status, entities.StatusUploaded)
	}

	if len(videos.outbox) != 1 {
		t.Fatalf("outbox len = %d, want 1", len(videos.outbox))
	}
	var env messaging.Event
	if err := json.Unmarshal(videos.outbox[0].Payload, &env); err != nil {
		t.Fatalf("unmarshal outbox envelope: %v", err)
	}
	if env.EventName != eventVideoUploaded {
		t.Errorf("EventName = %q, want %q", env.EventName, eventVideoUploaded)
	}

	var payload videoUploadedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("unmarshal event payload: %v", err)
	}
	if payload.UserEmail != "uploader@example.com" {
		t.Errorf("UserEmail = %q, want %q", payload.UserEmail, "uploader@example.com")
	}
	if payload.VideoID != video.ID.String() {
		t.Errorf("VideoID = %q, want %q", payload.VideoID, video.ID.String())
	}
}

func TestUploadVideo_UploadFails(t *testing.T) {
	users := newFakeUserRepository()
	videos := newFakeVideoRepository()
	uploader := newFakeUploader()
	uploader.uploadErr = errors.New("storage unavailable")

	user := entities.User{ID: uuid.New(), Email: "uploader@example.com"}
	if err := users.Create(context.Background(), &user); err != nil {
		t.Fatalf("setup Create user error = %v", err)
	}

	uc := NewUploadVideoUseCase(videos, users, uploader, "fiapx-videos")
	body := strings.NewReader("fake video bytes")
	_, err := uc.UploadVideo(context.Background(), user.ID, "myvideo.mp4", "video/mp4", int64(body.Len()), body)
	if err == nil {
		t.Fatal("expected error when storage upload fails")
	}
	if len(videos.videos) != 0 {
		t.Errorf("expected no video row to be saved on upload failure, got %d", len(videos.videos))
	}
}

func TestUploadVideo_UnknownUser(t *testing.T) {
	users := newFakeUserRepository()
	videos := newFakeVideoRepository()
	uploader := newFakeUploader()

	uc := NewUploadVideoUseCase(videos, users, uploader, "fiapx-videos")
	body := strings.NewReader("fake video bytes")
	_, err := uc.UploadVideo(context.Background(), uuid.New(), "myvideo.mp4", "video/mp4", int64(body.Len()), body)
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
}
