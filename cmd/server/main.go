package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/application/usecases"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/config"
	infradb "github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/db"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/storage"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/handlers"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(sqlDB *sql.DB) error {
	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	gormDB, err := infradb.Connect(cfg.DBDSN)
	if err != nil {
		slog.Error("failed to connect to db", "error", err)
		os.Exit(1)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		slog.Error("failed to get sql.DB", "error", err)
		os.Exit(1)
	}
	if err := runMigrations(sqlDB); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	s3Client, err := storage.NewS3Client(ctx, cfg.MinioEndpoint, cfg.MinioPublicEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL)
	if err != nil {
		slog.Error("failed to build storage client", "error", err)
		os.Exit(1)
	}

	userRepo := infradb.NewUserRepository(gormDB)
	videoRepo := infradb.NewVideoRepository(gormDB)

	registerUserUC := usecases.NewRegisterUserUseCase(userRepo)
	loginUserUC := usecases.NewLoginUserUseCase(userRepo, cfg.JWTSecret)
	uploadVideoUC := usecases.NewUploadVideoUseCase(videoRepo, userRepo, s3Client, cfg.MinioBucket)
	listVideosUC := usecases.NewListVideosUseCase(videoRepo)
	getDownloadURLUC := usecases.NewGetDownloadURLUseCase(videoRepo, s3Client)

	authHandler := handlers.NewAuthHandler(registerUserUC, loginUserUC)
	videoHandler := handlers.NewVideoHandler(uploadVideoUC, listVideosUC, getDownloadURLUC, gormDB)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Correlation(), middleware.Logging())
	r.MaxMultipartMemory = 32 << 20

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/register", authHandler.Register)
		v1.POST("/auth/login", authHandler.Login)

		videos := v1.Group("/videos")
		videos.Use(middleware.AuthRequired(cfg.JWTSecret))
		{
			videos.POST("", videoHandler.UploadVideo)
			videos.GET("", videoHandler.ListVideos)
			videos.GET("/:id/download", videoHandler.GetDownloadURL)
		}
	}
	r.GET("/healthz", videoHandler.Healthz)
	r.GET("/readyz", videoHandler.Readyz)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()
	slog.Info("server started", "port", cfg.Port)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}
