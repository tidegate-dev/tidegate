package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// ErrInvalidProjectName means a project name doesn't start with a lowercase
// letter, has something other than lowercase letters, digits and dashes, ends
// with a dash, or has more than 63 characters.
var ErrInvalidProjectName = errors.New("the project name must start with a lowercase letter, go on with lowercase letters, digits and dashes, " +
	"end with a letter or digit, and have at most 63 characters")

// RFC 1035 gives a DNS label this form, and a Kubernetes namespace has it too.
var projectNamePattern = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateProjectName returns ErrInvalidProjectName if name doesn't start with
// a lowercase letter, has something other than lowercase letters, digits and
// dashes, ends with a dash, or has more than 63 characters.
func ValidateProjectName(name string) error {
	if !projectNamePattern.MatchString(name) {
		return ErrInvalidProjectName
	}

	return nil
}

// ProjectLister lists projects.
type ProjectLister interface {
	// All returns every project, ordered by name.
	All(ctx context.Context) ([]*model.Project, error)
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
// ErrInvalidProjectName for a name that ValidateProjectName refuses, and
// model.ErrProjectNameTaken if another project has the name.
func (p *Projects) Create(ctx context.Context, name string) (*model.Project, error) {
	err := ValidateProjectName(name)
	if err != nil {
		return nil, err
	}

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
	return p.projects.All(ctx)
}
