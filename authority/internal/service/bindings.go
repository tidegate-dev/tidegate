package service

import (
	"context"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// RoleFinder finds a role in a project by the role's name.
type RoleFinder interface {
	// FindByProjectIDAndName ignores letter case and returns
	// model.ErrRoleNotFound if the project has no such role.
	FindByProjectIDAndName(ctx context.Context, projectID model.ProjectID, name string) (*model.Role, error)
}

// BindingLister lists the bindings of a project.
type BindingLister interface {
	// ListForProjectID orders the bindings by role name and then by user name.
	ListForProjectID(ctx context.Context, projectID model.ProjectID) ([]*model.BindingWithNames, error)
}

// Bindings creates and lists bindings.
type Bindings struct {
	projects ProjectFinder
	roles    RoleFinder
	users    UserFinder
	bindings BindingLister
	units    model.UnitFactory
}

// NewBindings returns Bindings.
func NewBindings(
	projects ProjectFinder,
	roles RoleFinder,
	users UserFinder,
	bindings BindingLister,
	units model.UnitFactory,
) *Bindings {
	return &Bindings{projects: projects, roles: roles, users: users, bindings: bindings, units: units}
}

// Create gives a role of a project to a user. It doesn't check the form of the
// names, because the API has done that.
func (b *Bindings) Create(
	ctx context.Context,
	projectName string,
	roleName string,
	userName string,
) (*model.BindingWithNames, error) {
	project, err := b.projects.FindByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	role, err := b.roles.FindByProjectIDAndName(ctx, project.ID, roleName)
	if err != nil {
		return nil, err
	}

	user, err := b.users.FindByName(ctx, userName)
	if err != nil {
		return nil, err
	}

	id, err := model.NewBindingID()
	if err != nil {
		return nil, err
	}

	unit := b.units.NewUnit()
	unit.Bindings().Add(&model.Binding{ID: id, ProjectID: project.ID, RoleID: role.ID, UserID: user.ID})

	err = unit.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &model.BindingWithNames{ID: id, ProjectID: project.ID, RoleName: role.Name, UserName: user.Name}, nil
}

// List returns the bindings of a project, ordered by role name and then by
// user name.
func (b *Bindings) List(ctx context.Context, projectName string) ([]*model.BindingWithNames, error) {
	project, err := b.projects.FindByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	return b.bindings.ListForProjectID(ctx, project.ID)
}
