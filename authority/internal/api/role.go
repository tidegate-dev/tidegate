package api

import (
	"context"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// RoleService creates roles.
type RoleService struct {
	roles *service.Roles
}

// NewRoleService returns a RoleService.
func NewRoleService(roles *service.Roles) *RoleService {
	return &RoleService{roles: roles}
}

// CreateRole creates a role in a project and returns it.
func (s *RoleService) CreateRole(
	ctx context.Context,
	req *connect.Request[authorityv1.CreateRoleRequest],
) (*connect.Response[authorityv1.CreateRoleResponse], error) {
	role, err := s.roles.Create(ctx, req.Msg.GetProject(), req.Msg.GetName(), permissions(req.Msg.GetPermissions()))
	if err != nil {
		return nil, clientError(err)
	}

	return connect.NewResponse(&authorityv1.CreateRoleResponse{Role: roleMessage(role)}), nil
}

func permissions(message *authorityv1.ProjectPermissions) model.ProjectPermissions {
	return model.ProjectPermissions{
		CreateRoles:    message.GetCreateRoles(),
		Escalate:       message.GetEscalate(),
		CreateBindings: message.GetCreateBindings(),
		Bind:           message.GetBind(),
	}
}

func roleMessage(role *model.Role) *authorityv1.Role {
	return &authorityv1.Role{
		Id:        role.ID.String(),
		ProjectId: role.ProjectID.String(),
		Name:      role.Name,
		Permissions: &authorityv1.ProjectPermissions{
			CreateRoles:    role.Permissions.CreateRoles,
			Escalate:       role.Permissions.Escalate,
			CreateBindings: role.Permissions.CreateBindings,
			Bind:           role.Permissions.Bind,
		},
	}
}
