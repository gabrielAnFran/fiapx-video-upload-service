package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	DBDSN              string
	AMQPURL            string
	DispatchIntervalMS int
	JWTSecret          string
	MinioEndpoint      string
	MinioAccessKey     string
	MinioSecretKey     string
	MinioBucket        string
	MinioUseSSL        bool
}

func Load() Config {
	return Config{
		Port:               getEnv("UPLOAD_PORT", "8081"),
		DBDSN:              getEnv("UPLOAD_DB_DSN", "host=localhost user=postgres password=postgres dbname=upload_service port=5432 sslmode=disable"),
		AMQPURL:            getEnv("UPLOAD_AMQP_URL", "amqp://guest:guest@localhost:5672/"),
		DispatchIntervalMS: getEnvInt("UPLOAD_DISPATCH_INTERVAL_MS", 500),
		JWTSecret:          getEnv("UPLOAD_JWT_SECRET", "dev-secret-change-me"),
		MinioEndpoint:      getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:     getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:        getEnv("MINIO_BUCKET", "fiapx-videos"),
		MinioUseSSL:        getEnvBool("MINIO_USE_SSL", false),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
