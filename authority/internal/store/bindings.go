package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// BindingReader queries the bindings table on the connection pool.
type BindingReader struct {
	queries *sqlc.Queries
}

// ByID returns the binding with the ID, or model.ErrBindingNotFound.
func (b *BindingReader) ByID(ctx context.Context, id model.BindingID) (*model.Binding, error) {
	row, err := b.queries.GetBinding(ctx, databaseID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrBindingNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("reading the binding: %w", err)
	}

	return &model.Binding{
		ID:        model.BindingID(row.ID.Bytes),
		ProjectID: model.ProjectID(row.ProjectID.Bytes),
		RoleID:    model.RoleID(row.RoleID.Bytes),
		UserID:    model.UserID(row.UserID.Bytes),
	}, nil
}

// InProject returns the bindings of the project with the names of their role
// and user. It orders them by the role name and then the user name, without
// regard to letter case.
func (b *BindingReader) InProject(ctx context.Context, projectID model.ProjectID) ([]*model.BindingWithNames, error) {
	rows, err := b.queries.ListBindingsOfProject(ctx, databaseID(projectID))
	if err != nil {
		return nil, fmt.Errorf("reading the bindings: %w", err)
	}

	bindings := make([]*model.BindingWithNames, len(rows))

	for i, row := range rows {
		bindings[i] = &model.BindingWithNames{
			ID:       model.BindingID(row.ID.Bytes),
			RoleName: row.RoleName,
			UserName: row.UserName,
		}
	}

	return bindings, nil
}

// BindingReadWriter belongs to a unit of work. It records writes, which the
// unit runs at Commit.
type BindingReadWriter struct {
	unit *Unit
}

// Add records the insert of binding. Commit returns model.ErrRoleNotFound if
// the project has no role with the ID, and model.ErrUserNotFound if the user
// doesn't exist.
func (b *BindingReadWriter) Add(binding *model.Binding) {
	b.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		err := queries.InsertBinding(ctx, sqlc.InsertBindingParams{
			ID:        databaseID(binding.ID),
			ProjectID: databaseID(binding.ProjectID),
			RoleID:    databaseID(binding.RoleID),
			UserID:    databaseID(binding.UserID),
		})

		switch {
		case isForeignKeyViolationOf(err, "bindings_project_id_role_id_fkey"):
			return model.ErrRoleNotFound
		case isForeignKeyViolationOf(err, "bindings_user_id_fkey"):
			return model.ErrUserNotFound
		case err != nil:
			return fmt.Errorf("creating the binding: %w", err)
		default:
			return nil
		}
	})
}

// Remove records the delete of binding. A row that is already gone counts as
// removed.
func (b *BindingReadWriter) Remove(binding *model.Binding) {
	b.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		err := queries.DeleteBinding(ctx, databaseID(binding.ID))
		if err != nil {
			return fmt.Errorf("deleting the binding: %w", err)
		}

		return nil
	})
}
