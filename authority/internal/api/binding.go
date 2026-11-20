package api

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
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
	binding, err := s.bindings.Create(ctx, req.Msg.GetProject(), req.Msg.GetRole(), req.Msg.GetUser())
	if err != nil {
		return nil, bindingError(err)
	}

	return connect.NewResponse(&authorityv1.CreateBindingResponse{Binding: bindingMessage(binding)}), nil
}

// ListBindings returns every binding of a project.
func (s *BindingService) ListBindings(
	ctx context.Context,
	req *connect.Request[authorityv1.ListBindingsRequest],
) (*connect.Response[authorityv1.ListBindingsResponse], error) {
	bindings, err := s.bindings.List(ctx, req.Msg.GetProject())
	if err != nil {
		return nil, bindingError(err)
	}

	messages := make([]*authorityv1.Binding, len(bindings))
	for i, binding := range bindings {
		messages[i] = bindingMessage(binding)
	}

	return connect.NewResponse(&authorityv1.ListBindingsResponse{Bindings: messages}), nil
}

// bindingError tells a caller that a user doesn't exist, which clientError
// keeps from everyone so that a login can't reveal it.
func bindingError(err error) error {
	if errors.Is(err, model.ErrUserNotFound) {
		return connect.NewError(connect.CodeNotFound, model.ErrUserNotFound)
	}

	return clientError(err)
}

func bindingMessage(binding *model.BindingWithNames) *authorityv1.Binding {
	return &authorityv1.Binding{
		Id:        binding.ID.String(),
		ProjectId: binding.ProjectID.String(),
		Role:      binding.RoleName,
		User:      binding.UserName,
	}
}
