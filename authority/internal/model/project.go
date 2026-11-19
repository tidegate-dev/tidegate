package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	// ErrProjectNotFound means no project has the given ID or name.
	ErrProjectNotFound = errors.New("project not found")
	// ErrProjectNameTaken means another project already has the name. Nothing
	// was written.
	ErrProjectNameTaken = errors.New("the project name is taken")
)

// ProjectID identifies a project.
type ProjectID uuid.UUID

// NewProjectID returns the ID for a new project.
func NewProjectID() (ProjectID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return ProjectID{}, fmt.Errorf("creating a project ID: %w", err)
	}

	return ProjectID(id), nil
}

// String returns the ID in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (id ProjectID) String() string {
	return uuid.UUID(id).String()
}

// Project holds assets, roles and bindings.
type Project struct {
	ID   ProjectID
	Name string
}

// ProjectReadWriter belongs to a Unit. It records writes, which the unit runs
// at Commit.
type ProjectReadWriter interface {
	// Add records the insert of project. Commit returns ErrProjectNameTaken
	// if another project has the name, ignoring letter case.
	Add(project *Project)
}
