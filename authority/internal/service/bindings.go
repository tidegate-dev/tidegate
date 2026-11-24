package service

import (
	"context"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// RoleFinder finds a role of a project by its name.
type RoleFinder interface {
	// FindByProjectIDAndName returns the role of the project with the name,
	// ignoring letter case, or model.ErrRoleNotFound.
	FindByProjectIDAndName(ctx context.Context, projectID model.ProjectID, name string) (*model.Role, error)
}

// BindingLister lists the bindings of a project.
type BindingLister interface {
	// ListForProjectID returns the bindings of the project with the names of
	// their role and user, ordered by the role name and then the user name.
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

// NewBindings returns Bindings that find projects, roles and users in
// projects, roles and users, read bindings from bindings and write in units of
// work from units.
func NewBindings(
	projects ProjectFinder,
	roles RoleFinder,
	users UserFinder,
	bindings BindingLister,
	units model.UnitFactory,
) *Bindings {
	return &Bindings{projects: projects, roles: roles, users: users, bindings: bindings, units: units}
}

// Create binds the role named roleName of the project named projectName to
// the user named userName and returns the binding. It returns
// model.ErrProjectNotFound, model.ErrRoleNotFound or model.ErrUserNotFound if
// the project, the role or the user doesn't exist, and model.ErrBindingExists
// if the user already has the role. The API checks the form of the project
// name and the role name before it calls Create.
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

// List returns the bindings of the project named projectName, ordered by the
// role name and then the user name. It returns model.ErrProjectNotFound if no
// project has that name.
func (b *Bindings) List(ctx context.Context, projectName string) ([]*model.BindingWithNames, error) {
	project, err := b.projects.FindByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	return b.bindings.ListForProjectID(ctx, project.ID)
}
