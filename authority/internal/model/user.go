// Package model holds the authority's data as plain Go types, without database types.
package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/phemera-dev/phemera/internal/secret"
)

var (
	// ErrUserNotFound means no user has the given ID or name.
	ErrUserNotFound = errors.New("user not found")
	// ErrUsersExist means a user already existed.
	ErrUsersExist = errors.New("a user already exists")
	// ErrUserNameTaken means another user already has the name. Nothing was
	// written.
	ErrUserNameTaken = errors.New("the user name is taken")
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
	// Version changes with every write. The store compares it at Commit.
	Version uint32
}

// UserReadWriter belongs to a Unit. It reads users right away and records
// writes, which the unit runs at Commit.
type UserReadWriter interface {
	// FindByName returns the user with the name, ignoring letter case, or
	// ErrUserNotFound.
	FindByName(ctx context.Context, name string) (*User, error)
	// AnyExist reports whether a user exists.
	AnyExist(ctx context.Context) (bool, error)
	// Add records the insert of user. Commit returns ErrUserNameTaken if
	// another user has the name.
	Add(user *User)
	// AddIfNoneExist records the insert of user as the first user. Commit
	// returns ErrUsersExist if a user exists by then.
	AddIfNoneExist(user *User)
	// Update records the update of every field of user. Commit returns
	// ErrChangedSinceRead if the user changed since it was read,
	// ErrUserNameTaken if another user has the name, and ErrUserNotFound if
	// it's gone.
	Update(user *User)
	// Remove records the delete of user. Commit returns ErrChangedSinceRead if
	// the user changed since it was read. A user that is already gone counts
	// as removed.
	Remove(user *User)
}
