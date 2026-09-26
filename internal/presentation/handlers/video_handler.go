package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/application/usecases"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/dto"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VideoHandler struct {
	uploadVideo    *usecases.UploadVideoUseCase
	listVideos     *usecases.ListVideosUseCase
	getDownloadURL *usecases.GetDownloadURLUseCase
	db             *gorm.DB
}

func NewVideoHandler(uploadVideo *usecases.UploadVideoUseCase, listVideos *usecases.ListVideosUseCase, getDownloadURL *usecases.GetDownloadURLUseCase, db *gorm.DB) *VideoHandler {
	return &VideoHandler{uploadVideo: uploadVideo, listVideos: listVideos, getDownloadURL: getDownloadURL, db: db}
}

func userIDFromContext(c *gin.Context) (uuid.UUID, error) {
	v, ok := c.Get(middleware.UserIDKey)
	if !ok {
		return uuid.UUID{}, errors.New("missing user id in context")
	}
	s, ok := v.(string)
	if !ok {
		return uuid.UUID{}, errors.New("user id in context is not a string")
	}
	return uuid.Parse(s)
}

func (h *VideoHandler) UploadVideo(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		problem(c, http.StatusUnauthorized, "Unauthorized", "invalid authenticated user")
		return
	}

	fileHeader, err := c.FormFile("video")
	if err != nil {
		problem(c, http.StatusBadRequest, "Bad Request", "missing or invalid \"video\" multipart field")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		problem(c, http.StatusBadRequest, "Bad Request", "failed to open uploaded file")
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	video, err := h.uploadVideo.UploadVideo(c.Request.Context(), userID, fileHeader.Filename, contentType, fileHeader.Size, file)
	if err != nil {
		slog.Error("upload video failed", "error", err, "user_id", userID)
		problem(c, http.StatusInternalServerError, "Internal Server Error", "failed to upload video")
		return
	}

	c.JSON(http.StatusAccepted, dto.UploadVideoResponse{VideoID: video.ID.String(), Status: string(video.Status)})
}

func (h *VideoHandler) ListVideos(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		problem(c, http.StatusUnauthorized, "Unauthorized", "invalid authenticated user")
		return
	}

	var status *entities.VideoStatus
	if v := c.Query("status"); v != "" {
		if !entities.IsValidVideoStatus(v) {
			problem(c, http.StatusBadRequest, "Bad Request", "unknown status value")
			return
		}
		s := entities.VideoStatus(v)
		status = &s
	}

	limit := 0
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			problem(c, http.StatusBadRequest, "Bad Request", "limit must be a positive integer")
			return
		}
		limit = n
	}

	videos, nextCursor, err := h.listVideos.ListVideos(c.Request.Context(), userID, status, c.Query("cursor"), limit)
	if err != nil {
		problem(c, http.StatusBadRequest, "Bad Request", "invalid pagination cursor")
		return
	}

	resp := dto.VideoListResponse{Videos: make([]dto.VideoResponse, 0, len(videos)), NextCursor: nextCursor}
	for i := range videos {
		resp.Videos = append(resp.Videos, toVideoResponse(&videos[i]))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *VideoHandler) GetDownloadURL(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		problem(c, http.StatusUnauthorized, "Unauthorized", "invalid authenticated user")
		return
	}

	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		problem(c, http.StatusBadRequest, "Bad Request", "id must be a valid UUID")
		return
	}

	url, err := h.getDownloadURL.GetDownloadURL(c.Request.Context(), userID, videoID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrNotFound):
			problem(c, http.StatusNotFound, "Not Found", "video not found")
		case errors.Is(err, repositories.ErrForbidden):
			problem(c, http.StatusForbidden, "Forbidden", "you do not have access to this video")
		case errors.Is(err, repositories.ErrNotReady):
			problem(c, http.StatusBadRequest, "Bad Request", "video is not ready for download")
		default:
			problem(c, http.StatusInternalServerError, "Internal Server Error", "failed to build download url")
		}
		return
	}

	c.JSON(http.StatusOK, dto.DownloadURLResponse{DownloadURL: url})
}

func (h *VideoHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *VideoHandler) Readyz(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func toVideoResponse(v *entities.Video) dto.VideoResponse {
	resp := dto.VideoResponse{
		VideoID:          v.ID.String(),
		Status:           string(v.Status),
		OriginalFilename: v.OriginalFilename,
		CreatedAt:        v.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		ErrorMessage:     v.ErrorMessage,
	}
	if v.CompletedAt != nil {
		s := v.CompletedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		resp.CompletedAt = &s
	}
	if v.FailedAt != nil {
		s := v.FailedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		resp.FailedAt = &s
	}
	return resp
}
