package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/application/usecases"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/dto"
	"github.com/gin-gonic/gin"
)

func newAuthTestRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/register", h.Register)
	router.POST("/auth/login", h.Login)
	return router
}

func doJSONRequest(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if s, ok := body.(string); ok {
		reader = bytes.NewReader([]byte(s))
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestAuthHandler_Register_Success(t *testing.T) {
	users := newFakeUserRepository()
	h := NewAuthHandler(usecases.NewRegisterUserUseCase(users), usecases.NewLoginUserUseCase(users, "secret"))
	router := newAuthTestRouter(h)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    "new@example.com",
		"password": "s3cr3t",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp dto.RegisterResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", resp.Email, "new@example.com")
	}
	if resp.UserID == "" {
		t.Error("expected non-empty UserID")
	}
}

func TestAuthHandler_Register_BadJSON(t *testing.T) {
	users := newFakeUserRepository()
	h := NewAuthHandler(usecases.NewRegisterUserUseCase(users), usecases.NewLoginUserUseCase(users, "secret"))
	router := newAuthTestRouter(h)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", "{not-json")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	assertProblem(t, rec, http.StatusBadRequest, "Bad Request")
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	users := newFakeUserRepository()
	h := NewAuthHandler(usecases.NewRegisterUserUseCase(users), usecases.NewLoginUserUseCase(users, "secret"))
	router := newAuthTestRouter(h)

	body := map[string]string{"email": "dup@example.com", "password": "s3cr3t"}
	if rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", body); rec.Code != http.StatusCreated {
		t.Fatalf("first register status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	assertProblem(t, rec, http.StatusConflict, "Conflict")
}

func TestAuthHandler_Register_RepositoryError(t *testing.T) {
	users := newFakeUserRepository()
	users.createErr = errUnexpected
	h := NewAuthHandler(usecases.NewRegisterUserUseCase(users), usecases.NewLoginUserUseCase(users, "secret"))
	router := newAuthTestRouter(h)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    "boom@example.com",
		"password": "s3cr3t",
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	assertProblem(t, rec, http.StatusInternalServerError, "Internal Server Error")
}

func TestAuthHandler_Login_Success(t *testing.T) {
	users := newFakeUserRepository()
	registerUC := usecases.NewRegisterUserUseCase(users)
	if _, err := registerUC.RegisterUser(context.Background(), "login@example.com", "s3cr3t"); err != nil {
		t.Fatalf("setup RegisterUser() error = %v", err)
	}
	h := NewAuthHandler(registerUC, usecases.NewLoginUserUseCase(users, "secret"))
	router := newAuthTestRouter(h)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "login@example.com",
		"password": "s3cr3t",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp dto.LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthHandler_Login_BadJSON(t *testing.T) {
	users := newFakeUserRepository()
	h := NewAuthHandler(usecases.NewRegisterUserUseCase(users), usecases.NewLoginUserUseCase(users, "secret"))
	router := newAuthTestRouter(h)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", "{not-json")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	users := newFakeUserRepository()
	h := NewAuthHandler(usecases.NewRegisterUserUseCase(users), usecases.NewLoginUserUseCase(users, "secret"))
	router := newAuthTestRouter(h)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "ghost@example.com",
		"password": "whatever",
	})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	assertProblem(t, rec, http.StatusUnauthorized, "Unauthorized")
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantTitle string) {
	t.Helper()
	var problem dto.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal problem body: %v (body = %s)", err, rec.Body.String())
	}
	if problem.Status != wantStatus {
		t.Errorf("problem.Status = %d, want %d", problem.Status, wantStatus)
	}
	if !strings.EqualFold(problem.Title, wantTitle) {
		t.Errorf("problem.Title = %q, want %q", problem.Title, wantTitle)
	}
}
