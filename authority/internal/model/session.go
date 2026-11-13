package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

// ErrSessionNotFound means no session that hasn't expired has the given token
// hash.
var ErrSessionNotFound = errors.New("session not found")

// SessionID identifies a session.
type SessionID uuid.UUID

// NewSessionID returns the ID for a new session.
func NewSessionID() (SessionID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return SessionID{}, fmt.Errorf("creating a session ID: %w", err)
	}

	return SessionID(id), nil
}

// String returns the ID in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (id SessionID) String() string {
	return uuid.UUID(id).String()
}

// Session is one login of a user.
type Session struct {
	ID     SessionID
	UserID UserID
	// TokenHash is the SHA-256 hash of the session token, as 32 raw bytes.
	TokenHash secret.String
	CreatedAt time.Time
	ExpiresAt time.Time
	// Version is the xmin of the row in Postgres at the time it was read or written.
	Version uint32
}
