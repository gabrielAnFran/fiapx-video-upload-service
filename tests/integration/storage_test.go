//go:build integration

package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestS3Client builds a storage.S3Client pointed at the shared MinIO
// container. Endpoint and publicEndpoint are the same value here since,
// unlike a docker-compose deployment, both the test process and the
// container are reachable at the same host:port with no in-network vs.
// host-network split.
func newTestS3Client(t *testing.T) *storage.S3Client {
	t.Helper()
	client, err := storage.NewS3Client(context.Background(), testMinIOEndpoint, testMinIOEndpoint, testMinIOAccessKey, testMinIOSecretKey, testMinIOBucket, false)
	require.NoError(t, err)
	return client
}

func TestS3Client_Upload_PersistsObject(t *testing.T) {
	client := newTestS3Client(t)
	ctx := context.Background()

	key := "uploads/" + uuid.NewString() + ".mp4"
	content := []byte("fake mp4 bytes for testing")
	require.NoError(t, client.Upload(ctx, key, bytes.NewReader(content), "video/mp4"))
}

func TestS3Client_PresignGetObject_DownloadsUploadedBytes(t *testing.T) {
	client := newTestS3Client(t)
	ctx := context.Background()

	key := "uploads/" + uuid.NewString() + ".mp4"
	content := []byte("the quick brown fox jumps over the lazy dog")
	require.NoError(t, client.Upload(ctx, key, bytes.NewReader(content), "video/mp4"))

	url, err := client.PresignGetObject(ctx, key, 5*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestS3Client_PresignGetObject_MissingKey_URLReturns404(t *testing.T) {
	client := newTestS3Client(t)
	ctx := context.Background()

	url, err := client.PresignGetObject(ctx, "uploads/does-not-exist.mp4", 5*time.Minute)
	require.NoError(t, err)

	resp, err := http.Get(url) //nolint:gosec // test-only URL to the ephemeral MinIO container
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
