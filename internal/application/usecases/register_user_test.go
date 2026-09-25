package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/auth"
)

func TestRegisterUser_Success(t *testing.T) {
	users := newFakeUserRepository()
	uc := NewRegisterUserUseCase(users)

	user, err := uc.RegisterUser(context.Background(), "new@example.com", "s3cr3t")
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "new@example.com")
	}
	if user.PasswordHash == "s3cr3t" {
		t.Fatal("password hash should not equal the plaintext password")
	}
	if err := auth.CheckPassword(user.PasswordHash, "s3cr3t"); err != nil {
		t.Errorf("stored hash does not match plaintext password: %v", err)
	}
}

func TestRegisterUser_EmptyEmail(t *testing.T) {
	users := newFakeUserRepository()
	uc := NewRegisterUserUseCase(users)

	_, err := uc.RegisterUser(context.Background(), "", "s3cr3t")
	if !errors.Is(err, ErrEmailRequired) {
		t.Errorf("error = %v, want %v", err, ErrEmailRequired)
	}
}

func TestRegisterUser_EmptyPassword(t *testing.T) {
	users := newFakeUserRepository()
	uc := NewRegisterUserUseCase(users)

	_, err := uc.RegisterUser(context.Background(), "new@example.com", "")
	if !errors.Is(err, ErrPasswordRequired) {
		t.Errorf("error = %v, want %v", err, ErrPasswordRequired)
	}
}

func TestRegisterUser_DuplicateEmail(t *testing.T) {
	users := newFakeUserRepository()
	uc := NewRegisterUserUseCase(users)

	_, err := uc.RegisterUser(context.Background(), "dup@example.com", "s3cr3t")
	if err != nil {
		t.Fatalf("first RegisterUser() error = %v", err)
	}

	_, err = uc.RegisterUser(context.Background(), "dup@example.com", "other-pass")
	if !errors.Is(err, repositories.ErrDuplicateEmail) {
		t.Errorf("error = %v, want %v", err, repositories.ErrDuplicateEmail)
	}
}
