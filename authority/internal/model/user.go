// Package model holds the authority's data as plain Go types, without database types.
package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

var (
	// ErrUserNotFound means no user has the given ID or name.
	ErrUserNotFound = errors.New("user not found")
	// ErrUsersExist means the users table already held a user.
	ErrUsersExist = errors.New("users already exist")
)

// UserID identifies a user.
type UserID uuid.UUID

// NewUserID returns the ID for a new user.
func NewUserID() (UserID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return UserID{}, fmt.Errorf("creating a user ID: %w", err)
	}

	return UserID(id), nil
}

// String returns the ID in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (id UserID) String() string {
	return uuid.UUID(id).String()
}

// User is a person who can sign in to the authority.
type User struct {
	ID   UserID
	Name string
	// PasswordHash is nil for a user without a local password.
	PasswordHash *secret.String
	// Version is the xmin of the row in Postgres at the time it was read or written.
	Version uint32
}
