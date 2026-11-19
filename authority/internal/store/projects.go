package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// ProjectReader queries the projects table on the connection pool.
type ProjectReader struct {
	queries *sqlc.Queries
}

// ByName returns the project with the name, ignoring letter case, or
// model.ErrProjectNotFound.
func (p *ProjectReader) ByName(ctx context.Context, name string) (*model.Project, error) {
	row, err := p.queries.GetProjectByName(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrProjectNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("reading the project: %w", err)
	}

	return &model.Project{ID: model.ProjectID(row.ID.Bytes), Name: row.Name}, nil
}

// All returns every project, ordered by name without regard to letter case.
func (p *ProjectReader) All(ctx context.Context) ([]*model.Project, error) {
	rows, err := p.queries.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the projects: %w", err)
	}

	projects := make([]*model.Project, len(rows))
	for i, row := range rows {
		projects[i] = &model.Project{ID: model.ProjectID(row.ID.Bytes), Name: row.Name}
	}

	return projects, nil
}

// ProjectReadWriter belongs to a unit of work. It records writes, which the
// unit runs at Commit.
type ProjectReadWriter struct {
	unit *Unit
}

// Add records the insert of project. Commit returns
// model.ErrProjectNameTaken if another project has the name, ignoring letter
// case.
func (p *ProjectReadWriter) Add(project *model.Project) {
	p.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		err := queries.InsertProject(ctx, sqlc.InsertProjectParams{
			ID:   databaseID(project.ID),
			Name: project.Name,
		})
		if isUniqueViolationOf(err, "projects_name_key") {
			return model.ErrProjectNameTaken
		}

		if err != nil {
			return fmt.Errorf("creating the project: %w", err)
		}

		return nil
	})
}
