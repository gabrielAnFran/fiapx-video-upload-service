package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/google/uuid"
)

func TestListVideos_Success(t *testing.T) {
	videos := newFakeVideoRepository()
	uc := NewListVideosUseCase(videos)

	userID := uuid.New()
	other := uuid.New()
	v1 := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusUploaded}
	v2 := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusCompleted}
	v3 := entities.Video{ID: uuid.New(), UserID: other, Status: entities.StatusUploaded}
	for _, v := range []entities.Video{v1, v2, v3} {
		if err := videos.Save(context.Background(), &v, nil); err != nil {
			t.Fatalf("setup Save() error = %v", err)
		}
	}

	got, cursor, err := uc.ListVideos(context.Background(), userID, nil, "", 0)
	if err != nil {
		t.Fatalf("ListVideos() error = %v", err)
	}
	if cursor != "" {
		t.Errorf("cursor = %q, want empty", cursor)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	for _, v := range got {
		if v.UserID != userID {
			t.Errorf("got video for user %v, want only %v", v.UserID, userID)
		}
	}
}

func TestListVideos_FiltersByStatus(t *testing.T) {
	videos := newFakeVideoRepository()
	uc := NewListVideosUseCase(videos)

	userID := uuid.New()
	uploaded := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusUploaded}
	completed := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusCompleted}
	for _, v := range []entities.Video{uploaded, completed} {
		if err := videos.Save(context.Background(), &v, nil); err != nil {
			t.Fatalf("setup Save() error = %v", err)
		}
	}

	status := entities.StatusCompleted
	got, _, err := uc.ListVideos(context.Background(), userID, &status, "", 0)
	if err != nil {
		t.Fatalf("ListVideos() error = %v", err)
	}

	// The fake repository ignores Status/Cursor/Limit filtering, but this
	// confirms the use case forwards them unchanged to the repository.
	if videos.lastFilter.Status == nil || *videos.lastFilter.Status != entities.StatusCompleted {
		t.Errorf("repository received Status = %v, want %v", videos.lastFilter.Status, entities.StatusCompleted)
	}
	if len(got) == 0 {
		t.Fatal("expected at least the videos owned by the user")
	}
}

func TestListVideos_PassesCursorAndLimit(t *testing.T) {
	videos := newFakeVideoRepository()
	uc := NewListVideosUseCase(videos)

	userID := uuid.New()
	_, nextCursor, err := uc.ListVideos(context.Background(), userID, nil, "some-cursor", 10)
	if err != nil {
		t.Fatalf("ListVideos() error = %v", err)
	}
	if nextCursor != videos.nextCursor {
		t.Errorf("nextCursor = %q, want %q", nextCursor, videos.nextCursor)
	}
	if videos.lastFilter.Cursor != "some-cursor" {
		t.Errorf("repository received Cursor = %q, want %q", videos.lastFilter.Cursor, "some-cursor")
	}
	if videos.lastFilter.Limit != 10 {
		t.Errorf("repository received Limit = %d, want %d", videos.lastFilter.Limit, 10)
	}
}

func TestListVideos_RepositoryError(t *testing.T) {
	videos := newFakeVideoRepository()
	videos.listErr = errors.New("query failed")
	uc := NewListVideosUseCase(videos)

	_, _, err := uc.ListVideos(context.Background(), uuid.New(), nil, "", 0)
	if err == nil {
		t.Fatal("expected error when repository List fails")
	}
	if !errors.Is(err, videos.listErr) {
		t.Errorf("error = %v, want %v", err, videos.listErr)
	}
}
