package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/auth"
	"github.com/google/uuid"
)

var (
	ErrEmailRequired    = errors.New("email is required")
	ErrPasswordRequired = errors.New("password is required")
)

type RegisterUserUseCase struct {
	users repositories.UserRepository
}

func NewRegisterUserUseCase(users repositories.UserRepository) *RegisterUserUseCase {
	return &RegisterUserUseCase{users: users}
}

// RegisterUser validates the input, hashes the password with bcrypt, and
// persists a new user. It maps a unique-email violation from the repository
// through to repositories.ErrDuplicateEmail so callers can respond 409.
func (uc *RegisterUserUseCase) RegisterUser(ctx context.Context, email, password string) (*entities.User, error) {
	if email == "" {
		return nil, ErrEmailRequired
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &entities.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.users.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			return nil, repositories.ErrDuplicateEmail
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
