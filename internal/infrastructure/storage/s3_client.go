package storage

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps an S3-compatible object storage client (MinIO in this
// service's deployment) for streamed uploads and presigned downloads.
type S3Client struct {
	client *s3.Client
	// presignClient is a second client, identical to client except for its
	// BaseEndpoint, used ONLY to mint presigned URLs. Presigning computes an
	// AWS SigV4 signature over the request's Host header, so a URL signed
	// against the internal endpoint (e.g. "minio:9000", the docker-compose
	// service name) cannot later have its host swapped for a
	// host-reachable one (e.g. "localhost:9000") — that invalidates the
	// signature and MinIO returns 403. Handing out a URL with the internal
	// hostname is just as broken, since nothing outside the docker network
	// can resolve it. Building the presign client against a distinct public
	// endpoint from the start avoids both problems. It never performs
	// network I/O itself (PresignGetObject only computes a signed URL
	// locally), so a second client here has no runtime cost.
	presignClient *s3.Client
	bucket        string
}

// NewS3Client builds an S3 client pointed at a MinIO endpoint using static
// credentials and path-style addressing (required by MinIO). publicEndpoint
// is the host:port used only when minting presigned URLs (see S3Client's
// presignClient field); pass the same value as endpoint when the internal
// and externally-reachable addresses coincide.
func NewS3Client(ctx context.Context, endpoint, publicEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Client, error) {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(scheme + "://" + endpoint)
	})

	presignClient := client
	if publicEndpoint != endpoint {
		presignClient = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.BaseEndpoint = aws.String(scheme + "://" + publicEndpoint)
		})
	}

	return &S3Client{client: client, presignClient: presignClient, bucket: bucket}, nil
}

// Upload streams body to the given object key using a multipart-safe
// uploader, so arbitrary-size files never get buffered in memory.
func (c *S3Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	uploader := manager.NewUploader(c.client)
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	return err
}

// PresignGetObject returns a time-limited URL that grants direct download
// access to the given object key.
func (c *S3Client) PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.presignClient)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
