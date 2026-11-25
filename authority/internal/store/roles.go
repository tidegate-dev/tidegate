package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phemera-dev/phemera/authority/internal/db/sqlc"
	"github.com/phemera-dev/phemera/authority/internal/model"
)

// RoleReader queries the roles table on the connection pool.
type RoleReader struct {
	queries *sqlc.Queries
}

// FindByProjectIDAndName returns the role of the project with the name,
// ignoring letter case, or model.ErrRoleNotFound.
func (r *RoleReader) FindByProjectIDAndName(ctx context.Context, projectID model.ProjectID, name string) (*model.Role, error) {
	row, err := r.queries.GetRoleByName(ctx, sqlc.GetRoleByNameParams{
		ProjectID: databaseID(projectID),
		Name:      name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrRoleNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("reading the role: %w", err)
	}

	return &model.Role{
		ID:        model.RoleID(row.ID.Bytes),
		ProjectID: model.ProjectID(row.ProjectID.Bytes),
		Name:      row.Name,
		Permissions: model.ProjectPermissions{
			CreateRoles:    row.CreateRoles,
			Escalate:       row.Escalate,
			CreateBindings: row.CreateBindings,
			Bind:           row.Bind,
		},
	}, nil
}

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
		case isUniqueViolationOf(err, "roles_name_unique_in_project"):
			return model.ErrRoleNameTaken
		case isForeignKeyViolationOf(err, "roles_project_fk"):
			return model.ErrProjectNotFound
		case err != nil:
			return fmt.Errorf("creating the role: %w", err)
		default:
			return nil
		}
	})
}
