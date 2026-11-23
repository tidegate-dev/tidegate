package api

import (
	"context"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// ProjectService creates and lists projects.
type ProjectService struct {
	projects *service.Projects
}

// NewProjectService returns a ProjectService.
func NewProjectService(projects *service.Projects) *ProjectService {
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

	return connect.NewResponse(&authorityv1.CreateProjectResponse{Id: project.ID.String(), Name: project.Name}), nil
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

	messages := make([]*authorityv1.ListProjectsResponse_Project, len(projects))
	for i, project := range projects {
		messages[i] = &authorityv1.ListProjectsResponse_Project{Id: project.ID.String(), Name: project.Name}
	}

	return connect.NewResponse(&authorityv1.ListProjectsResponse{Projects: messages}), nil
}
