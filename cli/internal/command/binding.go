package command

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/phemera-dev/phemera/cli/internal/connections"
	authorityv1 "github.com/phemera-dev/phemera/gen/phemera/authority/v1"
)

type bindingResult struct {
	ID        string `json:"id"        yaml:"id"`
	ProjectID string `json:"projectId" yaml:"projectId"`
	RoleName  string `json:"roleName"  yaml:"roleName"`
	UserName  string `json:"userName"  yaml:"userName"`
}

func newBindingResult(response *authorityv1.CreateBindingResponse) bindingResult {
	return bindingResult{
		ID:        response.GetId(),
		ProjectID: response.GetProjectId(),
		RoleName:  response.GetRoleName(),
		UserName:  response.GetUserName(),
	}
}

func (result bindingResult) String() string {
	return strings.Join(result.tableRow(), "\t")
}

func (result bindingResult) tableRow() []string {
	return []string{
		quoteIfUnprintable(result.ID),
		quoteIfUnprintable(result.RoleName),
		quoteIfUnprintable(result.UserName),
	}
}

type bindingsResult struct {
	Bindings []bindingResult `json:"bindings" yaml:"bindings"`
}

func newBindingsResult(bindings []*authorityv1.ListBindingsResponse_Binding) bindingsResult {
	result := bindingsResult{Bindings: make([]bindingResult, 0, len(bindings))}
	for _, binding := range bindings {
		result.Bindings = append(result.Bindings, bindingResult{
			ID:        binding.GetId(),
			ProjectID: binding.GetProjectId(),
			RoleName:  binding.GetRoleName(),
			UserName:  binding.GetUserName(),
		})
	}

	return result
}

func (result bindingsResult) String() string {
	if len(result.Bindings) == 0 {
		return "No bindings in the project. Run `phemera binding create` to give a role to a user."
	}

	rows := make([][]string, 0, len(result.Bindings))
	for _, binding := range result.Bindings {
		rows = append(rows, binding.tableRow())
	}

	return tableText([]string{"ID", "ROLE", "USER"}, rows)
}

func bindingCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "binding",
		Usage: "manage role bindings in a project",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "assign a role to a user inside the project and print the binding ID, the role and the user",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "project-name",
						Usage:    "name of the project the binding is added to",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "role-name",
						Usage:    "name of the role the binding is referencing",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "user-name",
						Usage:    "name of the user subject to the binding",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return bindingCreate(ctx, cmd, store)
				},
			},
			{
				Name:  "list",
				Usage: "print the ID, the role and the user name of every binding inside the project",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "project-name",
						Usage:    "name of the project whose bindings are listed",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return bindingList(ctx, cmd, store)
				},
			},
		},
	}
}

func bindingCreate(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 0 {
		return errors.New("want no arguments")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Binding().CreateBinding(ctx, connect.NewRequest(&authorityv1.CreateBindingRequest{
		ProjectName: cmd.String("project-name"),
		RoleName:    cmd.String("role-name"),
		UserName:    cmd.String("user-name"),
	}))
	if err != nil {
		return callFailed("creating the binding", connection, err)
	}

	return printResult(cmd, newBindingResult(response.Msg))
}

func bindingList(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 0 {
		return errors.New("want no arguments")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Binding().ListBindings(ctx, connect.NewRequest(&authorityv1.ListBindingsRequest{
		ProjectName: cmd.String("project-name"),
	}))
	if err != nil {
		return callFailed("listing the bindings", connection, err)
	}

	return printResult(cmd, newBindingsResult(response.Msg.GetBindings()))
}
