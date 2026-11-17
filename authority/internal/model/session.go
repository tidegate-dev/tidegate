package model

import (
	"context"
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
	TokenHash secret.Bytes
	CreatedAt time.Time
	ExpiresAt time.Time
	// Version is the xmin of the row in Postgres at the time it was read or written.
	Version uint32
}

// SessionReadWriter belongs to a Unit. It reads sessions right away and
// records writes, which the unit runs at Commit.
type SessionReadWriter interface {
	// ActiveByTokenHash returns the session with the token hash that expires
	// after now, or ErrSessionNotFound.
	ActiveByTokenHash(ctx context.Context, tokenHash secret.Bytes, now time.Time) (*Session, error)
	// Add records the insert of session. Commit returns ErrConflict if
	// another session has the token hash, and ErrUserNotFound if the user of
	// the session is gone.
	Add(session *Session)
	// Remove records the delete of session. Commit returns ErrConflict if the
	// session changed since it was read. A session that is already gone
	// counts as removed.
	Remove(session *Session)
	// RemoveOfUser records the delete of every session of the user.
	RemoveOfUser(userID UserID)
	// RemoveExpiredOfUser records the delete of every session of the user
	// that expires at now or earlier.
	RemoveExpiredOfUser(userID UserID, now time.Time)
}
