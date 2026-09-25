package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/auth"
	"github.com/gin-gonic/gin"
)

func newAuthRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", AuthRequired(secret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString(UserIDKey)})
	})
	return r
}

func TestAuthRequired_MissingHeader(t *testing.T) {
	r := newAuthRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthRequired_MalformedHeader(t *testing.T) {
	r := newAuthRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "NotBearer sometoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	r := newAuthRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthRequired_ValidToken(t *testing.T) {
	r := newAuthRouter("secret")
	token, err := auth.GenerateToken("user-123", "user@example.com", "secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if got := w.Body.String(); got != `{"user_id":"user-123"}` {
		t.Errorf("body = %q, want %q", got, `{"user_id":"user-123"}`)
	}
}
