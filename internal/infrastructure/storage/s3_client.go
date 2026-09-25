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
	bucket string
}

// NewS3Client builds an S3 client pointed at a MinIO endpoint using static
// credentials and path-style addressing (required by MinIO).
func NewS3Client(ctx context.Context, endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Client, error) {
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

	return &S3Client{client: client, bucket: bucket}, nil
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
	presignClient := s3.NewPresignClient(c.client)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
