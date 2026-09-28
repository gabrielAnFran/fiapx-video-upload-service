package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLogging_EmitsRequestLogLine(t *testing.T) {
	var buf strings.Builder
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prevLogger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Correlation(), Logging())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	out := buf.String()
	if !strings.Contains(out, "request") {
		t.Errorf("log output = %q, want it to contain %q", out, "request")
	}
	if !strings.Contains(out, "path=/ping") {
		t.Errorf("log output = %q, want it to contain path=/ping", out)
	}
	if !strings.Contains(out, "status=200") {
		t.Errorf("log output = %q, want it to contain status=200", out)
	}
	if !strings.Contains(out, "correlation_id=") {
		t.Errorf("log output = %q, want it to contain correlation_id=", out)
	}
}
