package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/auth"
)

func TestLoginUser_Success(t *testing.T) {
	users := newFakeUserRepository()
	registerUC := NewRegisterUserUseCase(users)
	if _, err := registerUC.RegisterUser(context.Background(), "login@example.com", "s3cr3t"); err != nil {
		t.Fatalf("setup RegisterUser() error = %v", err)
	}

	loginUC := NewLoginUserUseCase(users, "test-secret")
	token, err := loginUC.LoginUser(context.Background(), "login@example.com", "s3cr3t")
	if err != nil {
		t.Fatalf("LoginUser() error = %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := auth.ValidateToken(token, "test-secret")
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.Email != "login@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "login@example.com")
	}
}

func TestLoginUser_UnknownEmail(t *testing.T) {
	users := newFakeUserRepository()
	loginUC := NewLoginUserUseCase(users, "test-secret")

	_, err := loginUC.LoginUser(context.Background(), "ghost@example.com", "whatever")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestLoginUser_WrongPassword(t *testing.T) {
	users := newFakeUserRepository()
	registerUC := NewRegisterUserUseCase(users)
	if _, err := registerUC.RegisterUser(context.Background(), "login2@example.com", "correct-pass"); err != nil {
		t.Fatalf("setup RegisterUser() error = %v", err)
	}

	loginUC := NewLoginUserUseCase(users, "test-secret")
	_, err := loginUC.LoginUser(context.Background(), "login2@example.com", "wrong-pass")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("error = %v, want %v", err, ErrInvalidCredentials)
	}
}
