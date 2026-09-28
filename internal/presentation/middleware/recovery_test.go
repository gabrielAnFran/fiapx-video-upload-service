package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/dto"
	"github.com/gin-gonic/gin"
)

func TestRecovery_RecoversFromPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery())
	router.GET("/boom", func(c *gin.Context) {
		panic("something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()

	// If Recovery did not catch the panic, ServeHTTP itself would panic and
	// fail this test process instead of returning normally.
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var problem dto.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if problem.Status != http.StatusInternalServerError {
		t.Errorf("problem.Status = %d, want %d", problem.Status, http.StatusInternalServerError)
	}
	if problem.Title != "Internal Server Error" {
		t.Errorf("problem.Title = %q, want %q", problem.Title, "Internal Server Error")
	}
	if problem.Instance != "/boom" {
		t.Errorf("problem.Instance = %q, want %q", problem.Instance, "/boom")
	}
}

func TestRecovery_NoPanicPassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery())
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "fine")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "fine" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "fine")
	}
}
