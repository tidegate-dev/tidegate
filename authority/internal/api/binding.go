package api

import (
	"context"

	"connectrpc.com/connect"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/service"
	authorityv1 "github.com/phemera-dev/phemera/gen/phemera/authority/v1"
)

// BindingService creates and lists bindings.
type BindingService struct {
	bindings *service.Bindings
}

// NewBindingService returns a BindingService.
func NewBindingService(bindings *service.Bindings) *BindingService {
	return &BindingService{bindings: bindings}
}

// CreateBinding gives a role of a project to a user and returns the binding.
func (s *BindingService) CreateBinding(
	ctx context.Context,
	req *connect.Request[authorityv1.CreateBindingRequest],
) (*connect.Response[authorityv1.CreateBindingResponse], error) {
	binding, err := s.bindings.Create(ctx, req.Msg.GetProjectName(), req.Msg.GetRoleName(), req.Msg.GetUserName())
	if err != nil {
		return nil, clientError(err)
	}

	return connect.NewResponse(newCreateBindingResponse(binding)), nil
}

// ListBindings returns every binding of a project.
func (s *BindingService) ListBindings(
	ctx context.Context,
	req *connect.Request[authorityv1.ListBindingsRequest],
) (*connect.Response[authorityv1.ListBindingsResponse], error) {
	bindings, err := s.bindings.List(ctx, req.Msg.GetProjectName())
	if err != nil {
		return nil, clientError(err)
	}

	messages := make([]*authorityv1.ListBindingsResponse_Binding, len(bindings))
	for i, binding := range bindings {
		messages[i] = newListedBinding(binding)
	}

	return connect.NewResponse(&authorityv1.ListBindingsResponse{Bindings: messages}), nil
}

func newCreateBindingResponse(binding *model.BindingWithNames) *authorityv1.CreateBindingResponse {
	return &authorityv1.CreateBindingResponse{
		Id:        binding.ID.String(),
		ProjectId: binding.ProjectID.String(),
		RoleName:  binding.RoleName,
		UserName:  binding.UserName,
	}
}

func newListedBinding(binding *model.BindingWithNames) *authorityv1.ListBindingsResponse_Binding {
	return &authorityv1.ListBindingsResponse_Binding{
		Id:        binding.ID.String(),
		ProjectId: binding.ProjectID.String(),
		RoleName:  binding.RoleName,
		UserName:  binding.UserName,
	}
}
