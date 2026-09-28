package handlers

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/entities"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/google/uuid"
)

// errUnexpected is a generic sentinel used across handler tests to simulate
// an unexpected repository/storage failure that should surface as a 500.
var errUnexpected = errors.New("unexpected failure")

// fakeUserRepository is an in-memory repositories.UserRepository, mirroring
// the fake used by the usecases package tests.
type fakeUserRepository struct {
	mu        sync.Mutex
	users     map[uuid.UUID]entities.User
	byEmail   map[string]uuid.UUID
	createErr error
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		users:   map[uuid.UUID]entities.User{},
		byEmail: map[string]uuid.UUID{},
	}
}

func (f *fakeUserRepository) Create(_ context.Context, u *entities.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, exists := f.byEmail[u.Email]; exists {
		return repositories.ErrDuplicateEmail
	}
	f.users[u.ID] = *u
	f.byEmail[u.Email] = u.ID
	return nil
}

func (f *fakeUserRepository) FindByEmail(_ context.Context, email string) (*entities.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byEmail[email]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	u := f.users[id]
	return &u, nil
}

func (f *fakeUserRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return &u, nil
}

// fakeVideoRepository is an in-memory repositories.VideoRepository.
type fakeVideoRepository struct {
	mu         sync.Mutex
	videos     map[uuid.UUID]entities.Video
	outbox     []repositories.OutboxEvent
	saveErr    error
	listErr    error
	nextCursor string
}

func newFakeVideoRepository() *fakeVideoRepository {
	return &fakeVideoRepository{videos: map[uuid.UUID]entities.Video{}}
}

func (f *fakeVideoRepository) Save(_ context.Context, v *entities.Video, outboxEvent *repositories.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.videos[v.ID] = *v
	if outboxEvent != nil {
		f.outbox = append(f.outbox, *outboxEvent)
	}
	return nil
}

func (f *fakeVideoRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.Video, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.videos[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return &v, nil
}

func (f *fakeVideoRepository) List(_ context.Context, filter repositories.VideoFilter) ([]entities.Video, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, "", f.listErr
	}
	out := make([]entities.Video, 0, len(f.videos))
	for _, v := range f.videos {
		if v.UserID == filter.UserID {
			out = append(out, v)
		}
	}
	return out, f.nextCursor, nil
}

// fakeUploader is an in-memory usecases.Uploader.
type fakeUploader struct {
	mu        sync.Mutex
	uploaded  map[string][]byte
	uploadErr error
}

func newFakeUploader() *fakeUploader {
	return &fakeUploader{uploaded: map[string][]byte{}}
}

func (f *fakeUploader) Upload(_ context.Context, key string, body io.Reader, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.uploadErr != nil {
		return f.uploadErr
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.uploaded[key] = b
	return nil
}

// fakePresigner is an in-memory usecases.Presigner.
type fakePresigner struct {
	presignErr error
}

func (f *fakePresigner) PresignGetObject(_ context.Context, key string, _ time.Duration) (string, error) {
	if f.presignErr != nil {
		return "", f.presignErr
	}
	return "https://minio.local/" + key + "?signed=1", nil
}
