package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrRoleNameTaken means another role of the project already has the name.
// Nothing was written.
var ErrRoleNameTaken = errors.New("the role name is taken")

// RoleID identifies a role.
type RoleID uuid.UUID

// NewRoleID returns the ID for a new role.
func NewRoleID() (RoleID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return RoleID{}, fmt.Errorf("creating a role ID: %w", err)
	}

	return RoleID(id), nil
}

// String returns the ID in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (id RoleID) String() string {
	return uuid.UUID(id).String()
}

// Role gives its holders permissions in one project.
type Role struct {
	ID          RoleID
	ProjectID   ProjectID
	Name        string
	Permissions ProjectPermissions
}

// ProjectPermissions says what the holders of a role may do with the roles
// and bindings of its project.
type ProjectPermissions struct {
	// CreateRoles means the holders may create roles in the project.
	CreateRoles bool
	// Escalate means the holders may create roles with permissions that they
	// don't have themselves.
	Escalate bool
	// CreateBindings means the holders may create bindings in the project.
	CreateBindings bool
	// Bind means the holders may bind roles with permissions that they don't
	// have themselves.
	Bind bool
}

// RoleReadWriter belongs to a Unit. It records writes, which the unit runs at
// Commit.
type RoleReadWriter interface {
	// Add records the insert of role. Commit returns ErrRoleNameTaken if
	// another role of the project has the name, ignoring letter case, and
	// ErrProjectNotFound if the project doesn't exist.
	Add(role *Role)
}
