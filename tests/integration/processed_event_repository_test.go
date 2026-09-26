//go:build integration

package integration

import (
	"context"
	"testing"

	infradb "github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessedEventRepository_IsProcessed_MarkProcessed(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	eventID := uuid.New()
	processed, err := repo.IsProcessed(ctx, eventID)
	require.NoError(t, err)
	assert.False(t, processed)

	require.NoError(t, repo.MarkProcessed(ctx, eventID))

	processed, err = repo.IsProcessed(ctx, eventID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestProcessedEventRepository_MarkProcessed_IdempotentOnConflict(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	repo := infradb.NewVideoRepository(db)
	ctx := context.Background()

	eventID := uuid.New()
	require.NoError(t, repo.MarkProcessed(ctx, eventID))
	// Marking the same event again must not error (DoNothing on conflict).
	require.NoError(t, repo.MarkProcessed(ctx, eventID))

	processed, err := repo.IsProcessed(ctx, eventID)
	require.NoError(t, err)
	assert.True(t, processed)
}
