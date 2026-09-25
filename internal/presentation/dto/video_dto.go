package dto

type UploadVideoResponse struct {
	VideoID string `json:"video_id"`
	Status  string `json:"status"`
}

type VideoResponse struct {
	VideoID          string  `json:"video_id"`
	Status           string  `json:"status"`
	OriginalFilename string  `json:"original_filename"`
	CreatedAt        string  `json:"created_at"`
	CompletedAt      *string `json:"completed_at,omitempty"`
	FailedAt         *string `json:"failed_at,omitempty"`
	ErrorMessage     *string `json:"error_message,omitempty"`
}

type VideoListResponse struct {
	Videos     []VideoResponse `json:"videos"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type DownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}
