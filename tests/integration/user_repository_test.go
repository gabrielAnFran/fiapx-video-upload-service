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
)

func newTestUser(email string) *entities.User {
	now := time.Now().UTC()
	return &entities.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "bcrypt-hash-placeholder",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestUserRepository_CreateAndFind(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	repo := infradb.NewUserRepository(db)
	ctx := context.Background()

	u := newTestUser("alice@example.com")
	require.NoError(t, repo.Create(ctx, u))

	byID, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.Email, byID.Email)
	assert.Equal(t, u.PasswordHash, byID.PasswordHash)

	byEmail, err := repo.FindByEmail(ctx, u.Email)
	require.NoError(t, err)
	assert.Equal(t, u.ID, byEmail.ID)
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	repo := infradb.NewUserRepository(db)
	ctx := context.Background()

	first := newTestUser("dup@example.com")
	require.NoError(t, repo.Create(ctx, first))

	second := newTestUser("dup@example.com")
	err := repo.Create(ctx, second)
	assert.ErrorIs(t, err, repositories.ErrDuplicateEmail)
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	repo := infradb.NewUserRepository(db)

	_, err := repo.FindByEmail(context.Background(), "missing@example.com")
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	truncateAll(t, db)
	repo := infradb.NewUserRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}
