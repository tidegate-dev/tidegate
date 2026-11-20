package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrBindingNotFound means no binding has the given ID.
var ErrBindingNotFound = errors.New("binding not found")

// BindingID identifies a binding.
type BindingID uuid.UUID

// NewBindingID returns the ID for a new binding.
func NewBindingID() (BindingID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return BindingID{}, fmt.Errorf("creating a binding ID: %w", err)
	}

	return BindingID(id), nil
}

// String returns the ID in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (id BindingID) String() string {
	return uuid.UUID(id).String()
}

// Binding gives a role of one project to a user until someone deletes it.
type Binding struct {
	ID        BindingID
	ProjectID ProjectID
	RoleID    RoleID
	UserID    UserID
}

// BindingWithNames is a binding in a list, with the names of its role and its
// user. It can't be written.
type BindingWithNames struct {
	ID        BindingID
	ProjectID ProjectID
	RoleName  string
	UserName  string
}

// BindingReadWriter belongs to a Unit. It records writes, which the unit runs
// at Commit.
type BindingReadWriter interface {
	// Add records the insert of binding. Commit returns ErrRoleNotFound if
	// the project has no role with the ID, and ErrUserNotFound if the user
	// doesn't exist.
	Add(binding *Binding)
	// Remove records the delete of binding. A binding that is already gone
	// counts as removed.
	Remove(binding *Binding)
}
