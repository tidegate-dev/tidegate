package service

import (
	"context"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// ProjectFinder finds a project by its name.
type ProjectFinder interface {
	// ByName returns the project with the name or model.ErrProjectNotFound.
	ByName(ctx context.Context, name string) (*model.Project, error)
}

// Roles creates roles.
type Roles struct {
	projects ProjectFinder
	units    model.UnitFactory
}

// NewRoles returns Roles that find projects in projects and write in units of
// work from units.
func NewRoles(projects ProjectFinder, units model.UnitFactory) *Roles {
	return &Roles{projects: projects, units: units}
}

// Create creates a role with the name and the permissions in the project
// named projectName and returns it. It returns model.ErrProjectNotFound if no
// project has that name, and model.ErrRoleNameTaken if another role of the
// project has the name. The API checks the form of both names before it calls
// Create.
func (r *Roles) Create(
	ctx context.Context,
	projectName string,
	name string,
	permissions model.ProjectPermissions,
) (*model.Role, error) {
	project, err := r.projects.ByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	id, err := model.NewRoleID()
	if err != nil {
		return nil, err
	}

	role := &model.Role{ID: id, ProjectID: project.ID, Name: name, Permissions: permissions}

	unit := r.units.NewUnit()
	unit.Roles().Add(role)

	err = unit.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return role, nil
}
