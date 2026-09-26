//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	infradb "github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createTestUserRow inserts a user directly (via UserRepository) so that
// videos referencing it via the user_id foreign key can be saved.
func createTestUserRow(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	u := newTestUser(uuid.NewString() + "@example.com")
	require.NoError(t, infradb.NewUserRepository(db).Create(context.Background(), u))
	return u.ID
}

func newTestVideo(userID uuid.UUID) *entities.Video {
	now := time.Now().UTC()
	return &entities.Video{
		ID:               uuid.New(),
		UserID:           userID,
		OriginalFilename: "movie.mp4",
		ContentType:      "video/mp4",
		SizeBytes:        1024,
		SourceBucket:     "uploads",
		SourceObjectKey:  "raw/" + uuid.NewString() + ".mp4",
		Status:           entities.StatusUploaded,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func TestVideoRepository_SaveAndFindByID(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	userID := createTestUserRow(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	v := newTestVideo(userID)
	require.NoError(t, repo.Save(ctx, v, nil))

	found, err := repo.FindByID(ctx, v.ID)
	require.NoError(t, err)
	assert.Equal(t, v.UserID, found.UserID)
	assert.Equal(t, entities.StatusUploaded, found.Status)
	assert.Equal(t, v.OriginalFilename, found.OriginalFilename)
}

func TestVideoRepository_FindByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	repo := infradb.NewVideoRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestVideoRepository_Save_UpsertsStatusOnConflict(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	userID := createTestUserRow(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	v := newTestVideo(userID)
	require.NoError(t, repo.Save(ctx, v, nil))

	// Transition the video to PROCESSING then to COMPLETED, saving the same
	// row (same ID) each time, mirroring how the worker would update status
	// as processing progresses.
	v.Status = entities.StatusProcessing
	v.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.Save(ctx, v, nil))

	processing, err := repo.FindByID(ctx, v.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.StatusProcessing, processing.Status)

	frameCount := 42
	zipBucket := "processed"
	zipKey := "zips/" + v.ID.String() + ".zip"
	completedAt := time.Now().UTC()
	v.Status = entities.StatusCompleted
	v.FrameCount = &frameCount
	v.ZipBucket = &zipBucket
	v.ZipObjectKey = &zipKey
	v.CompletedAt = &completedAt
	require.NoError(t, repo.Save(ctx, v, nil))

	completed, err := repo.FindByID(ctx, v.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.StatusCompleted, completed.Status)
	require.NotNil(t, completed.FrameCount)
	assert.Equal(t, 42, *completed.FrameCount)
	require.NotNil(t, completed.ZipBucket)
	assert.Equal(t, zipBucket, *completed.ZipBucket)
	require.NotNil(t, completed.CompletedAt)
}

func TestVideoRepository_Save_WithOutboxEvent(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	userID := createTestUserRow(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	v := newTestVideo(userID)
	outboxEvent := &repositories.OutboxEvent{
		EventID:     uuid.New(),
		AggregateID: v.ID,
		EventName:   "VideoUploaded",
		Payload:     []byte(`{"video_id":"` + v.ID.String() + `"}`),
		Headers:     []byte(`{}`),
	}
	require.NoError(t, repo.Save(ctx, v, outboxEvent))

	rows, err := repo.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "VideoUploaded", rows[0].EventName)
	assert.Equal(t, outboxEvent.EventID, rows[0].EventID)
}

func TestVideoRepository_List_FiltersAndPagination(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	userID := createTestUserRow(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		v := newTestVideo(userID)
		require.NoError(t, repo.Save(ctx, v, nil))
		time.Sleep(5 * time.Millisecond) // force distinct created_at for stable ordering
	}
	// A different user's video must be excluded by the filter.
	otherUserID := createTestUserRow(t, db)
	other := newTestVideo(otherUserID)
	require.NoError(t, repo.Save(ctx, other, nil))

	page1, cursor1, err := repo.List(ctx, repositories.VideoFilter{UserID: userID, Limit: 2})
	require.NoError(t, err)
	require.Len(t, page1, 2)
	require.NotEmpty(t, cursor1)

	page2, cursor2, err := repo.List(ctx, repositories.VideoFilter{UserID: userID, Limit: 2, Cursor: cursor1})
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Empty(t, cursor2)

	_, _, err = repo.List(ctx, repositories.VideoFilter{UserID: userID, Cursor: "not-valid-base64!!"})
	assert.Error(t, err)
}

func TestVideoRepository_List_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	userID := createTestUserRow(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	uploaded := newTestVideo(userID)
	require.NoError(t, repo.Save(ctx, uploaded, nil))

	completed := newTestVideo(userID)
	completed.Status = entities.StatusCompleted
	require.NoError(t, repo.Save(ctx, completed, nil))

	status := entities.StatusCompleted
	rows, _, err := repo.List(ctx, repositories.VideoFilter{UserID: userID, Status: &status})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, completed.ID, rows[0].ID)
}
