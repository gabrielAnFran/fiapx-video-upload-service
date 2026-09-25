package db

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type videoModel struct {
	ID               uuid.UUID  `gorm:"column:id;primaryKey"`
	UserID           uuid.UUID  `gorm:"column:user_id"`
	OriginalFilename string     `gorm:"column:original_filename"`
	ContentType      string     `gorm:"column:content_type"`
	SizeBytes        int64      `gorm:"column:size_bytes"`
	SourceBucket     string     `gorm:"column:source_bucket"`
	SourceObjectKey  string     `gorm:"column:source_object_key"`
	Status           string     `gorm:"column:status"`
	ZipBucket        *string    `gorm:"column:zip_bucket"`
	ZipObjectKey     *string    `gorm:"column:zip_object_key"`
	FrameCount       *int       `gorm:"column:frame_count"`
	ErrorMessage     *string    `gorm:"column:error_message"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	FailedAt         *time.Time `gorm:"column:failed_at"`
}

func (videoModel) TableName() string { return "videos" }

type outboxModel struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement"`
	EventID     uuid.UUID  `gorm:"column:event_id"`
	AggregateID uuid.UUID  `gorm:"column:aggregate_id"`
	EventName   string     `gorm:"column:event_name"`
	Payload     []byte     `gorm:"column:payload;type:jsonb"`
	Headers     []byte     `gorm:"column:headers;type:jsonb"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	PublishedAt *time.Time `gorm:"column:published_at"`
}

func (outboxModel) TableName() string { return "outbox" }

type processedEventModel struct {
	EventID     uuid.UUID `gorm:"column:event_id;primaryKey"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
}

func (processedEventModel) TableName() string { return "processed_events" }

// VideoRepository is a GORM-backed implementation of the domain repository
// interfaces defined in internal/domain/repositories.
type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func toVideoModel(v *entities.Video) videoModel {
	return videoModel{
		ID:               v.ID,
		UserID:           v.UserID,
		OriginalFilename: v.OriginalFilename,
		ContentType:      v.ContentType,
		SizeBytes:        v.SizeBytes,
		SourceBucket:     v.SourceBucket,
		SourceObjectKey:  v.SourceObjectKey,
		Status:           string(v.Status),
		ZipBucket:        v.ZipBucket,
		ZipObjectKey:     v.ZipObjectKey,
		FrameCount:       v.FrameCount,
		ErrorMessage:     v.ErrorMessage,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
		CompletedAt:      v.CompletedAt,
		FailedAt:         v.FailedAt,
	}
}

func fromVideoModel(m videoModel) entities.Video {
	return entities.Video{
		ID:               m.ID,
		UserID:           m.UserID,
		OriginalFilename: m.OriginalFilename,
		ContentType:      m.ContentType,
		SizeBytes:        m.SizeBytes,
		SourceBucket:     m.SourceBucket,
		SourceObjectKey:  m.SourceObjectKey,
		Status:           entities.VideoStatus(m.Status),
		ZipBucket:        m.ZipBucket,
		ZipObjectKey:     m.ZipObjectKey,
		FrameCount:       m.FrameCount,
		ErrorMessage:     m.ErrorMessage,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		CompletedAt:      m.CompletedAt,
		FailedAt:         m.FailedAt,
	}
}

func (r *VideoRepository) Save(ctx context.Context, v *entities.Video, outboxEvent *repositories.OutboxEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m := toVideoModel(v)
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"status", "zip_bucket", "zip_object_key", "frame_count",
				"error_message", "updated_at", "completed_at", "failed_at",
			}),
		}).Create(&m).Error; err != nil {
			return fmt.Errorf("save video: %w", err)
		}

		if outboxEvent != nil {
			om := outboxModel{
				EventID:     outboxEvent.EventID,
				AggregateID: outboxEvent.AggregateID,
				EventName:   outboxEvent.EventName,
				Payload:     outboxEvent.Payload,
				Headers:     outboxEvent.Headers,
				CreatedAt:   time.Now().UTC(),
			}
			if err := tx.Create(&om).Error; err != nil {
				return fmt.Errorf("save outbox event: %w", err)
			}
		}
		return nil
	})
}

func (r *VideoRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Video, error) {
	var m videoModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	v := fromVideoModel(m)
	return &v, nil
}

type videoCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func encodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw := fmt.Sprintf("%s|%s", createdAt.UTC().Format(time.RFC3339Nano), id.String())
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (*videoCursor, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	parts := splitOnce(string(raw), '|')
	if parts == nil {
		return nil, fmt.Errorf("malformed cursor")
	}
	createdAtStr, idStr := parts[0], parts[1]
	t, err := time.Parse(time.RFC3339Nano, createdAtStr)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	return &videoCursor{CreatedAt: t, ID: id}, nil
}

func splitOnce(s string, sep byte) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}

func (r *VideoRepository) List(ctx context.Context, filter repositories.VideoFilter) ([]entities.Video, string, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	q := r.db.WithContext(ctx).Model(&videoModel{}).Where("user_id = ?", filter.UserID)
	if filter.Status != nil {
		q = q.Where("status = ?", string(*filter.Status))
	}
	if filter.Cursor != "" {
		c, err := decodeCursor(filter.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
		q = q.Where("(created_at, id) < (?, ?)", c.CreatedAt, c.ID)
	}

	var rows []videoModel
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, "", err
	}

	out := make([]entities.Video, 0, len(rows))
	for _, m := range rows {
		out = append(out, fromVideoModel(m))
	}

	nextCursor := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return out, nextCursor, nil
}

// OutboxRepository implementation.

func (r *VideoRepository) FetchUnpublished(ctx context.Context, batch int) ([]repositories.OutboxRow, error) {
	var rows []outboxModel
	if err := r.db.WithContext(ctx).Where("published_at IS NULL").Order("created_at ASC").Limit(batch).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]repositories.OutboxRow, 0, len(rows))
	for _, m := range rows {
		out = append(out, repositories.OutboxRow{
			ID:        m.ID,
			EventID:   m.EventID,
			EventName: m.EventName,
			Payload:   m.Payload,
			Headers:   m.Headers,
		})
	}
	return out, nil
}

func (r *VideoRepository) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&outboxModel{}).Where("id IN ?", ids).Update("published_at", now).Error
}

// ProcessedEventRepository implementation.

func (r *VideoRepository) IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&processedEventModel{}).Where("event_id = ?", eventID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *VideoRepository) MarkProcessed(ctx context.Context, eventID uuid.UUID) error {
	m := processedEventModel{EventID: eventID, ProcessedAt: time.Now().UTC()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&m).Error
}
