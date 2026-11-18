package api

import (
	"context"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// Projects creates and lists projects.
type Projects interface {
	// Create creates a project with the name and returns it.
	Create(ctx context.Context, name string) (*model.Project, error)
	// List returns every project.
	List(ctx context.Context) ([]*model.Project, error)
}

// ProjectService creates and lists projects.
type ProjectService struct {
	projects Projects
}

// NewProjectService returns a ProjectService.
func NewProjectService(projects Projects) *ProjectService {
	return &ProjectService{projects: projects}
}

// CreateProject creates a project and returns it.
func (s *ProjectService) CreateProject(
	ctx context.Context,
	req *connect.Request[authorityv1.CreateProjectRequest],
) (*connect.Response[authorityv1.CreateProjectResponse], error) {
	project, err := s.projects.Create(ctx, req.Msg.GetName())
	if err != nil {
		return nil, clientError(err)
	}

	return connect.NewResponse(&authorityv1.CreateProjectResponse{Project: projectMessage(project)}), nil
}

// ListProjects returns every project.
func (s *ProjectService) ListProjects(
	ctx context.Context,
	_ *connect.Request[authorityv1.ListProjectsRequest],
) (*connect.Response[authorityv1.ListProjectsResponse], error) {
	projects, err := s.projects.List(ctx)
	if err != nil {
		return nil, err
	}

	messages := make([]*authorityv1.Project, len(projects))
	for i, project := range projects {
		messages[i] = projectMessage(project)
	}

	return connect.NewResponse(&authorityv1.ListProjectsResponse{Projects: messages}), nil
}

func projectMessage(project *model.Project) *authorityv1.Project {
	return &authorityv1.Project{Id: project.ID.String(), Name: project.Name}
}
