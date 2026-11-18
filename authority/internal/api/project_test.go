package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestProjectServiceCreateProjectReturnsTheNewProject(t *testing.T) {
	t.Parallel()

	// arrange
	projects := &fakeProjects{nextID: model.ProjectID{1}}
	projectService := NewProjectService(projects)
	request := connect.NewRequest(&authorityv1.CreateProjectRequest{Name: "payments"})

	// act
	response, err := projectService.CreateProject(t.Context(), request)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "01000000-0000-0000-0000-000000000000", response.Msg.GetProject().GetId())
	assert.Equal(t, "payments", response.Msg.GetProject().GetName())
}

func TestProjectServiceCreateProjectGivesATakenNameItsCode(t *testing.T) {
	t.Parallel()

	// arrange
	projectService := NewProjectService(&fakeProjects{createErr: model.ErrProjectNameTaken})
	request := connect.NewRequest(&authorityv1.CreateProjectRequest{Name: "payments"})

	// act
	_, err := projectService.CreateProject(t.Context(), request)

	// assert
	assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestProjectServiceListProjectsReturnsEveryProject(t *testing.T) {
	t.Parallel()

	// arrange
	projects := &fakeProjects{projects: []*model.Project{
		{ID: model.ProjectID{1}, Name: "payments"},
		{ID: model.ProjectID{2}, Name: "search"},
	}}
	projectService := NewProjectService(projects)
	request := connect.NewRequest(&authorityv1.ListProjectsRequest{})

	// act
	response, err := projectService.ListProjects(t.Context(), request)

	// assert
	require.NoError(t, err)

	names := []string{}
	for _, project := range response.Msg.GetProjects() {
		names = append(names, project.GetName())
	}

	assert.Equal(t, []string{"payments", "search"}, names)
}

type fakeProjects struct {
	nextID    model.ProjectID
	createErr error
	projects  []*model.Project
}

func (f *fakeProjects) Create(_ context.Context, name string) (*model.Project, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	project := &model.Project{ID: f.nextID, Name: name}
	f.projects = append(f.projects, project)

	return project, nil
}

func (f *fakeProjects) List(context.Context) ([]*model.Project, error) {
	return f.projects, nil
}
