//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	infradb "github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxRepository_FetchUnpublished_MarkPublished(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	userID := createTestUserRow(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	v := newTestVideo(userID)
	event := &repositories.OutboxEvent{
		EventID:     uuid.New(),
		AggregateID: v.ID,
		EventName:   "VideoUploaded",
		Payload:     []byte(`{"a":1}`),
		Headers:     []byte(`{}`),
	}
	require.NoError(t, repo.Save(ctx, v, event))

	rows, err := repo.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "VideoUploaded", rows[0].EventName)

	require.NoError(t, repo.MarkPublished(ctx, []int64{rows[0].ID}))
	require.NoError(t, repo.MarkPublished(ctx, nil)) // no-op path must not error

	rows2, err := repo.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, rows2)
}

func TestOutboxRepository_FetchUnpublished_RespectsBatchLimit(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	userID := createTestUserRow(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		v := newTestVideo(userID)
		event := &repositories.OutboxEvent{
			EventID:     uuid.New(),
			AggregateID: v.ID,
			EventName:   "VideoUploaded",
			Payload:     []byte(`{}`),
			Headers:     []byte(`{}`),
		}
		require.NoError(t, repo.Save(ctx, v, event))
	}

	rows, err := repo.FetchUnpublished(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}
