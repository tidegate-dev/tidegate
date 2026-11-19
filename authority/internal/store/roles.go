package store

import (
	"context"
	"fmt"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// RoleReadWriter belongs to a unit of work. It records writes, which the unit
// runs at Commit.
type RoleReadWriter struct {
	unit *Unit
}

// Add records the insert of role. Commit returns model.ErrRoleNameTaken if
// another role of the project has the name, ignoring letter case, and
// model.ErrProjectNotFound if the project doesn't exist.
func (r *RoleReadWriter) Add(role *model.Role) {
	r.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		err := queries.InsertRole(ctx, sqlc.InsertRoleParams{
			ID:             databaseID(role.ID),
			ProjectID:      databaseID(role.ProjectID),
			Name:           role.Name,
			CreateRoles:    role.Permissions.CreateRoles,
			Escalate:       role.Permissions.Escalate,
			CreateBindings: role.Permissions.CreateBindings,
			Bind:           role.Permissions.Bind,
		})
		switch {
		case isUniqueViolationOf(err, "roles_project_id_name_key"):
			return model.ErrRoleNameTaken
		case isForeignKeyViolation(err):
			return model.ErrProjectNotFound
		case err != nil:
			return fmt.Errorf("creating the role: %w", err)
		default:
			return nil
		}
	})
}
