package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"syscall"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/application/usecases"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/config"
	infradb "github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/db"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/infrastructure/messaging"
	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"os/signal"
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

	amqpConn, err := messaging.Dial(cfg.AMQPURL)
	if err != nil {
		slog.Error("failed to connect to amqp", "error", err)
		os.Exit(1)
	}
	defer amqpConn.Close()

	if _, err := amqpConn.DeclareServiceQueue("upload-service", []string{"video.status.completed", "video.status.failed"}); err != nil {
		slog.Error("failed to declare service queue", "error", err)
		os.Exit(1)
	}

	videoRepo := infradb.NewVideoRepository(gormDB)
	completedUC := usecases.NewHandleStatusCompletedUseCase(videoRepo, videoRepo)
	failedUC := usecases.NewHandleStatusFailedUseCase(videoRepo, videoRepo)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("worker started")

	handler := func(ctx context.Context, ev messaging.Event) error {
		switch ev.EventName {
		case "video.status.completed":
			return completedUC.Handle(ctx, ev)
		case "video.status.failed":
			return failedUC.Handle(ctx, ev)
		default:
			slog.Warn("ignoring unhandled event", "event", ev.EventName)
			return nil
		}
	}

	if err := amqpConn.Consume(ctx, "upload-service", handler); err != nil && ctx.Err() == nil {
		slog.Error("consumer stopped with error", "error", err)
		os.Exit(1)
	}
	slog.Info("worker stopped")
}
