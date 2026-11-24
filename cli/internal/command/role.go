package command

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

type permissionsResult struct {
	CreateRoles    bool `json:"createRoles"    yaml:"createRoles"`
	Escalate       bool `json:"escalate"       yaml:"escalate"`
	CreateBindings bool `json:"createBindings" yaml:"createBindings"`
	Bind           bool `json:"bind"           yaml:"bind"`
}

type roleResult struct {
	ID          string            `json:"id"          yaml:"id"`
	ProjectID   string            `json:"projectId"   yaml:"projectId"`
	Name        string            `json:"name"        yaml:"name"`
	Permissions permissionsResult `json:"permissions" yaml:"permissions"`
}

func newRoleResult(response *authorityv1.CreateRoleResponse) roleResult {
	return roleResult{
		ID:        response.GetId(),
		ProjectID: response.GetProjectId(),
		Name:      response.GetName(),
		Permissions: permissionsResult{
			CreateRoles:    response.GetPermissions().GetCreateRoles(),
			Escalate:       response.GetPermissions().GetEscalate(),
			CreateBindings: response.GetPermissions().GetCreateBindings(),
			Bind:           response.GetPermissions().GetBind(),
		},
	}
}

func (result roleResult) String() string {
	return quoteIfUnprintable(result.Name) + "\t" + quoteIfUnprintable(result.ID)
}

func roleCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "role",
		Usage: "create roles",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "create a role in a project and print its name and ID",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "project",
						Usage: "name of the project that gets the role",
					},
					&cli.BoolFlag{
						Name:  "create-roles",
						Usage: "the holders may create roles in the project",
					},
					&cli.BoolFlag{
						Name:  "escalate",
						Usage: "the holders may create roles with permissions that they don't have themselves",
					},
					&cli.BoolFlag{
						Name:  "create-bindings",
						Usage: "the holders may create bindings in the project",
					},
					&cli.BoolFlag{
						Name:  "bind",
						Usage: "the holders may bind roles with permissions that they don't have themselves",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return roleCreate(ctx, cmd, store)
				},
			},
		},
	}
}

func roleCreate(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one name")
	}

	if cmd.String("project") == "" {
		return errors.New("want a project, name it with --project")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Role().CreateRole(ctx, connect.NewRequest(&authorityv1.CreateRoleRequest{
		ProjectName: cmd.String("project"),
		Name:        cmd.Args().First(),
		Permissions: &authorityv1.ProjectPermissions{
			CreateRoles:    cmd.Bool("create-roles"),
			Escalate:       cmd.Bool("escalate"),
			CreateBindings: cmd.Bool("create-bindings"),
			Bind:           cmd.Bool("bind"),
		},
	}))
	if err != nil {
		return callFailed("creating the role", connection, err)
	}

	return printResult(cmd, newRoleResult(response.Msg))
}
