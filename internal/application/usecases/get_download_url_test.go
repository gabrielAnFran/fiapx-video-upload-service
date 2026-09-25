package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/google/uuid"
)

func TestGetDownloadURL_Success(t *testing.T) {
	videos := newFakeVideoRepository()
	presigner := &fakePresigner{}
	uc := NewGetDownloadURLUseCase(videos, presigner)

	userID := uuid.New()
	zipKey := "zips/some/frames.zip"
	video := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusCompleted, ZipObjectKey: &zipKey}
	if err := videos.Save(context.Background(), &video, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	url, err := uc.GetDownloadURL(context.Background(), userID, video.ID)
	if err != nil {
		t.Fatalf("GetDownloadURL() error = %v", err)
	}
	if url == "" {
		t.Error("expected non-empty download url")
	}
}

func TestGetDownloadURL_Forbidden(t *testing.T) {
	videos := newFakeVideoRepository()
	presigner := &fakePresigner{}
	uc := NewGetDownloadURLUseCase(videos, presigner)

	zipKey := "zips/some/frames.zip"
	video := entities.Video{ID: uuid.New(), UserID: uuid.New(), Status: entities.StatusCompleted, ZipObjectKey: &zipKey}
	if err := videos.Save(context.Background(), &video, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	_, err := uc.GetDownloadURL(context.Background(), uuid.New(), video.ID)
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Errorf("error = %v, want %v", err, repositories.ErrForbidden)
	}
}

func TestGetDownloadURL_NotReady(t *testing.T) {
	videos := newFakeVideoRepository()
	presigner := &fakePresigner{}
	uc := NewGetDownloadURLUseCase(videos, presigner)

	userID := uuid.New()
	video := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusProcessing}
	if err := videos.Save(context.Background(), &video, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	_, err := uc.GetDownloadURL(context.Background(), userID, video.ID)
	if !errors.Is(err, repositories.ErrNotReady) {
		t.Errorf("error = %v, want %v", err, repositories.ErrNotReady)
	}
}

func TestGetDownloadURL_VideoNotFound(t *testing.T) {
	videos := newFakeVideoRepository()
	presigner := &fakePresigner{}
	uc := NewGetDownloadURLUseCase(videos, presigner)

	_, err := uc.GetDownloadURL(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("error = %v, want %v", err, repositories.ErrNotFound)
	}
}
