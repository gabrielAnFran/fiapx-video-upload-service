package usecases

import (
	"context"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/google/uuid"
)

type ListVideosUseCase struct {
	videos repositories.VideoRepository
}

func NewListVideosUseCase(videos repositories.VideoRepository) *ListVideosUseCase {
	return &ListVideosUseCase{videos: videos}
}

// ListVideos is a thin wrapper over the repository, scoping every query to
// the given userID so callers can never list another user's videos.
func (uc *ListVideosUseCase) ListVideos(ctx context.Context, userID uuid.UUID, status *entities.VideoStatus, cursor string, limit int) ([]entities.Video, string, error) {
	return uc.videos.List(ctx, repositories.VideoFilter{
		UserID: userID,
		Status: status,
		Cursor: cursor,
		Limit:  limit,
	})
}
