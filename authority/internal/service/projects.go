package service

import (
	"context"

	"github.com/phemera-dev/phemera/authority/internal/model"
)

// ProjectLister lists projects.
type ProjectLister interface {
	// ListAll returns every project, ordered by name.
	ListAll(ctx context.Context) ([]*model.Project, error)
}

// Projects creates and lists projects.
type Projects struct {
	projects ProjectLister
	units    model.UnitFactory
}

// NewProjects returns Projects that read from projects and write in units of
// work from units.
func NewProjects(projects ProjectLister, units model.UnitFactory) *Projects {
	return &Projects{projects: projects, units: units}
}

// Create creates a project with the name and returns it. It returns
// model.ErrProjectNameTaken if another project has the name. The API checks
// the form of the name before it calls Create.
func (p *Projects) Create(ctx context.Context, name string) (*model.Project, error) {
	id, err := model.NewProjectID()
	if err != nil {
		return nil, err
	}

	project := &model.Project{ID: id, Name: name}

	unit := p.units.NewUnit()
	unit.Projects().Add(project)

	err = unit.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return project, nil
}

// List returns every project, ordered by name.
func (p *Projects) List(ctx context.Context) ([]*model.Project, error) {
	return p.projects.ListAll(ctx)
}
