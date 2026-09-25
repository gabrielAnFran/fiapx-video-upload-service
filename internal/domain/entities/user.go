package entities

import (
	"time"

	"github.com/google/uuid"
)

// User is a registered account able to upload videos and check their status.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
