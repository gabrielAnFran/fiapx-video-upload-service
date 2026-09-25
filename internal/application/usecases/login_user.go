package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/auth"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type LoginUserUseCase struct {
	users     repositories.UserRepository
	jwtSecret string
}

func NewLoginUserUseCase(users repositories.UserRepository, jwtSecret string) *LoginUserUseCase {
	return &LoginUserUseCase{users: users, jwtSecret: jwtSecret}
}

// LoginUser verifies the given credentials and, on success, issues a signed
// JWT for the user.
func (uc *LoginUserUseCase) LoginUser(ctx context.Context, email, password string) (string, error) {
	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("find user: %w", err)
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(user.ID.String(), user.Email, uc.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return token, nil
}
