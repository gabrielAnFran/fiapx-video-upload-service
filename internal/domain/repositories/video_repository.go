package repositories

import (
	"context"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/google/uuid"
)

type VideoFilter struct {
	UserID uuid.UUID
	Status *entities.VideoStatus
	Limit  int
	Cursor string
}

type VideoRepository interface {
	// Save persists the video row and, when outboxEvent is non-nil, writes it
	// to the outbox in the same transaction.
	Save(ctx context.Context, v *entities.Video, outboxEvent *OutboxEvent) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Video, error)
	List(ctx context.Context, filter VideoFilter) ([]entities.Video, string, error)
}

type OutboxEvent struct {
	EventID     uuid.UUID
	AggregateID uuid.UUID
	EventName   string
	Payload     []byte
	Headers     []byte
}

type OutboxRepository interface {
	FetchUnpublished(ctx context.Context, batch int) ([]OutboxRow, error)
	MarkPublished(ctx context.Context, ids []int64) error
}

type OutboxRow struct {
	ID        int64
	EventID   uuid.UUID
	EventName string
	Payload   []byte
	Headers   []byte
}

type ProcessedEventRepository interface {
	IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkProcessed(ctx context.Context, eventID uuid.UUID) error
}
