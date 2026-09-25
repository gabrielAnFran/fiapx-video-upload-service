package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("UPLOAD_PORT", "")
	t.Setenv("UPLOAD_DB_DSN", "")
	t.Setenv("UPLOAD_AMQP_URL", "")
	t.Setenv("UPLOAD_DISPATCH_INTERVAL_MS", "")
	t.Setenv("UPLOAD_JWT_SECRET", "")
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")
	t.Setenv("MINIO_BUCKET", "")
	t.Setenv("MINIO_USE_SSL", "")

	cfg := Load()

	if cfg.Port != "8081" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8081")
	}
	if cfg.DBDSN != "host=localhost user=postgres password=postgres dbname=upload_service port=5432 sslmode=disable" {
		t.Errorf("unexpected default DBDSN: %q", cfg.DBDSN)
	}
	if cfg.AMQPURL != "amqp://guest:guest@localhost:5672/" {
		t.Errorf("unexpected default AMQPURL: %q", cfg.AMQPURL)
	}
	if cfg.DispatchIntervalMS != 500 {
		t.Errorf("DispatchIntervalMS = %d, want 500", cfg.DispatchIntervalMS)
	}
	if cfg.JWTSecret != "dev-secret-change-me" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "dev-secret-change-me")
	}
	if cfg.MinioEndpoint != "localhost:9000" {
		t.Errorf("MinioEndpoint = %q, want %q", cfg.MinioEndpoint, "localhost:9000")
	}
	if cfg.MinioAccessKey != "minioadmin" {
		t.Errorf("MinioAccessKey = %q, want %q", cfg.MinioAccessKey, "minioadmin")
	}
	if cfg.MinioSecretKey != "minioadmin" {
		t.Errorf("MinioSecretKey = %q, want %q", cfg.MinioSecretKey, "minioadmin")
	}
	if cfg.MinioBucket != "fiapx-videos" {
		t.Errorf("MinioBucket = %q, want %q", cfg.MinioBucket, "fiapx-videos")
	}
	if cfg.MinioUseSSL != false {
		t.Errorf("MinioUseSSL = %v, want false", cfg.MinioUseSSL)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("UPLOAD_PORT", "9999")
	t.Setenv("UPLOAD_DB_DSN", "custom-dsn")
	t.Setenv("UPLOAD_AMQP_URL", "amqp://custom/")
	t.Setenv("UPLOAD_DISPATCH_INTERVAL_MS", "1500")
	t.Setenv("UPLOAD_JWT_SECRET", "custom-secret")
	t.Setenv("MINIO_ENDPOINT", "minio.example.com:9000")
	t.Setenv("MINIO_ACCESS_KEY", "custom-access")
	t.Setenv("MINIO_SECRET_KEY", "custom-secret-key")
	t.Setenv("MINIO_BUCKET", "custom-bucket")
	t.Setenv("MINIO_USE_SSL", "true")

	cfg := Load()

	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9999")
	}
	if cfg.DBDSN != "custom-dsn" {
		t.Errorf("DBDSN = %q, want %q", cfg.DBDSN, "custom-dsn")
	}
	if cfg.AMQPURL != "amqp://custom/" {
		t.Errorf("AMQPURL = %q, want %q", cfg.AMQPURL, "amqp://custom/")
	}
	if cfg.DispatchIntervalMS != 1500 {
		t.Errorf("DispatchIntervalMS = %d, want 1500", cfg.DispatchIntervalMS)
	}
	if cfg.JWTSecret != "custom-secret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "custom-secret")
	}
	if cfg.MinioEndpoint != "minio.example.com:9000" {
		t.Errorf("MinioEndpoint = %q, want %q", cfg.MinioEndpoint, "minio.example.com:9000")
	}
	if cfg.MinioAccessKey != "custom-access" {
		t.Errorf("MinioAccessKey = %q, want %q", cfg.MinioAccessKey, "custom-access")
	}
	if cfg.MinioSecretKey != "custom-secret-key" {
		t.Errorf("MinioSecretKey = %q, want %q", cfg.MinioSecretKey, "custom-secret-key")
	}
	if cfg.MinioBucket != "custom-bucket" {
		t.Errorf("MinioBucket = %q, want %q", cfg.MinioBucket, "custom-bucket")
	}
	if cfg.MinioUseSSL != true {
		t.Errorf("MinioUseSSL = %v, want true", cfg.MinioUseSSL)
	}
}

func TestLoad_InvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("UPLOAD_DISPATCH_INTERVAL_MS", "not-a-number")

	cfg := Load()

	if cfg.DispatchIntervalMS != 500 {
		t.Errorf("DispatchIntervalMS = %d, want default 500 on parse error", cfg.DispatchIntervalMS)
	}
}

func TestLoad_InvalidBoolFallsBackToDefault(t *testing.T) {
	t.Setenv("MINIO_USE_SSL", "not-a-bool")

	cfg := Load()

	if cfg.MinioUseSSL != false {
		t.Errorf("MinioUseSSL = %v, want default false on parse error", cfg.MinioUseSSL)
	}
}
