package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/application/usecases"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/dto"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setAuthContext injects a fake authenticated user id into the gin context,
// mimicking what middleware.AuthRequired does after validating a JWT.
func setAuthContext(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID != "" {
			c.Set(middleware.UserIDKey, userID)
		}
		c.Next()
	}
}

func newVideoTestHandler(t *testing.T) (*VideoHandler, *fakeVideoRepository, *fakeUserRepository, *fakeUploader, *fakePresigner) {
	t.Helper()
	users := newFakeUserRepository()
	videos := newFakeVideoRepository()
	uploader := newFakeUploader()
	presigner := &fakePresigner{}

	uploadUC := usecases.NewUploadVideoUseCase(videos, users, uploader, "fiapx-videos")
	listUC := usecases.NewListVideosUseCase(videos)
	downloadUC := usecases.NewGetDownloadURLUseCase(videos, presigner)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	h := NewVideoHandler(uploadUC, listUC, downloadUC, db)
	return h, videos, users, uploader, presigner
}

func newVideoTestRouter(h *VideoHandler, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(setAuthContext(userID))
	router.POST("/videos", h.UploadVideo)
	router.GET("/videos", h.ListVideos)
	router.GET("/videos/:id/download-url", h.GetDownloadURL)
	router.GET("/healthz", h.Healthz)
	router.GET("/readyz", h.Readyz)
	return router
}

func multipartVideoBody(t *testing.T, fieldName, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	if fieldName != "" {
		fw, err := w.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return buf, w.FormDataContentType()
}

func TestVideoHandler_UploadVideo_Success(t *testing.T) {
	h, videos, users, uploader, _ := newVideoTestHandler(t)

	user := entities.User{ID: uuid.New(), Email: "uploader@example.com"}
	if err := users.Create(context.Background(), &user); err != nil {
		t.Fatalf("setup Create user error = %v", err)
	}

	router := newVideoTestRouter(h, user.ID.String())
	body, contentType := multipartVideoBody(t, "video", "clip.mp4", "fake video bytes")

	req := httptest.NewRequest(http.MethodPost, "/videos", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var resp dto.UploadVideoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != string(entities.StatusUploaded) {
		t.Errorf("Status = %q, want %q", resp.Status, entities.StatusUploaded)
	}
	if len(uploader.uploaded) != 1 {
		t.Errorf("expected exactly 1 uploaded object, got %d", len(uploader.uploaded))
	}
	if len(videos.videos) != 1 {
		t.Errorf("expected exactly 1 saved video, got %d", len(videos.videos))
	}
}

func TestVideoHandler_UploadVideo_MissingAuth(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, "")
	body, contentType := multipartVideoBody(t, "video", "clip.mp4", "fake video bytes")

	req := httptest.NewRequest(http.MethodPost, "/videos", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestVideoHandler_UploadVideo_MissingFile(t *testing.T) {
	h, _, users, _, _ := newVideoTestHandler(t)
	user := entities.User{ID: uuid.New(), Email: "uploader@example.com"}
	if err := users.Create(context.Background(), &user); err != nil {
		t.Fatalf("setup Create user error = %v", err)
	}
	router := newVideoTestRouter(h, user.ID.String())

	body, contentType := multipartVideoBody(t, "", "", "")
	req := httptest.NewRequest(http.MethodPost, "/videos", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestVideoHandler_UploadVideo_UseCaseError(t *testing.T) {
	h, _, users, uploader, _ := newVideoTestHandler(t)
	uploader.uploadErr = errUnexpected

	user := entities.User{ID: uuid.New(), Email: "uploader@example.com"}
	if err := users.Create(context.Background(), &user); err != nil {
		t.Fatalf("setup Create user error = %v", err)
	}
	router := newVideoTestRouter(h, user.ID.String())

	body, contentType := multipartVideoBody(t, "video", "clip.mp4", "fake video bytes")
	req := httptest.NewRequest(http.MethodPost, "/videos", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestVideoHandler_ListVideos_Success(t *testing.T) {
	h, videos, _, _, _ := newVideoTestHandler(t)
	userID := uuid.New()
	v := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusUploaded, OriginalFilename: "a.mp4"}
	if err := videos.Save(context.Background(), &v, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}

	router := newVideoTestRouter(h, userID.String())
	req := httptest.NewRequest(http.MethodGet, "/videos", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp dto.VideoListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Videos) != 1 {
		t.Fatalf("len(Videos) = %d, want 1", len(resp.Videos))
	}
	if resp.Videos[0].VideoID != v.ID.String() {
		t.Errorf("VideoID = %q, want %q", resp.Videos[0].VideoID, v.ID.String())
	}
}

func TestVideoHandler_ListVideos_MissingAuth(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, "")
	req := httptest.NewRequest(http.MethodGet, "/videos", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestVideoHandler_ListVideos_InvalidStatus(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, uuid.New().String())
	req := httptest.NewRequest(http.MethodGet, "/videos?status=BOGUS", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestVideoHandler_ListVideos_ValidStatusFilter(t *testing.T) {
	h, videos, _, _, _ := newVideoTestHandler(t)
	userID := uuid.New()
	v := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusCompleted}
	if err := videos.Save(context.Background(), &v, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}
	router := newVideoTestRouter(h, userID.String())

	req := httptest.NewRequest(http.MethodGet, "/videos?status=COMPLETED&limit=5&cursor=abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestVideoHandler_ListVideos_InvalidLimit(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, uuid.New().String())
	req := httptest.NewRequest(http.MethodGet, "/videos?limit=-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestVideoHandler_ListVideos_RepositoryError(t *testing.T) {
	h, videos, _, _, _ := newVideoTestHandler(t)
	videos.listErr = errUnexpected
	router := newVideoTestRouter(h, uuid.New().String())

	req := httptest.NewRequest(http.MethodGet, "/videos", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestVideoHandler_GetDownloadURL_Success(t *testing.T) {
	h, videos, _, _, _ := newVideoTestHandler(t)
	userID := uuid.New()
	zipKey := "zips/some/frames.zip"
	v := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusCompleted, ZipObjectKey: &zipKey}
	if err := videos.Save(context.Background(), &v, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}
	router := newVideoTestRouter(h, userID.String())

	req := httptest.NewRequest(http.MethodGet, "/videos/"+v.ID.String()+"/download-url", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp dto.DownloadURLResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.DownloadURL == "" {
		t.Error("expected non-empty download url")
	}
}

func TestVideoHandler_GetDownloadURL_MissingAuth(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, "")

	req := httptest.NewRequest(http.MethodGet, "/videos/"+uuid.New().String()+"/download-url", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestVideoHandler_GetDownloadURL_InvalidID(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, uuid.New().String())

	req := httptest.NewRequest(http.MethodGet, "/videos/not-a-uuid/download-url", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestVideoHandler_GetDownloadURL_NotFound(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, uuid.New().String())

	req := httptest.NewRequest(http.MethodGet, "/videos/"+uuid.New().String()+"/download-url", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestVideoHandler_GetDownloadURL_Forbidden(t *testing.T) {
	h, videos, _, _, _ := newVideoTestHandler(t)
	zipKey := "zips/some/frames.zip"
	v := entities.Video{ID: uuid.New(), UserID: uuid.New(), Status: entities.StatusCompleted, ZipObjectKey: &zipKey}
	if err := videos.Save(context.Background(), &v, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}
	router := newVideoTestRouter(h, uuid.New().String())

	req := httptest.NewRequest(http.MethodGet, "/videos/"+v.ID.String()+"/download-url", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestVideoHandler_GetDownloadURL_NotReady(t *testing.T) {
	h, videos, _, _, _ := newVideoTestHandler(t)
	userID := uuid.New()
	v := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusProcessing}
	if err := videos.Save(context.Background(), &v, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}
	router := newVideoTestRouter(h, userID.String())

	req := httptest.NewRequest(http.MethodGet, "/videos/"+v.ID.String()+"/download-url", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestVideoHandler_GetDownloadURL_PresignError(t *testing.T) {
	h, videos, _, _, presigner := newVideoTestHandler(t)
	presigner.presignErr = errUnexpected
	userID := uuid.New()
	zipKey := "zips/some/frames.zip"
	v := entities.Video{ID: uuid.New(), UserID: userID, Status: entities.StatusCompleted, ZipObjectKey: &zipKey}
	if err := videos.Save(context.Background(), &v, nil); err != nil {
		t.Fatalf("setup Save() error = %v", err)
	}
	router := newVideoTestRouter(h, userID.String())

	req := httptest.NewRequest(http.MethodGet, "/videos/"+v.ID.String()+"/download-url", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestVideoHandler_Healthz(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, "")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestVideoHandler_Readyz_OK(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	router := newVideoTestRouter(h, "")

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestVideoHandler_Readyz_Unavailable(t *testing.T) {
	h, _, _, _, _ := newVideoTestHandler(t)
	sqlDB, err := h.db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql.DB: %v", err)
	}
	router := newVideoTestRouter(h, "")

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}
