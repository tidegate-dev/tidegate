package command

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

type bindingResult struct {
	ID        string `json:"id"        yaml:"id"`
	ProjectID string `json:"projectId" yaml:"projectId"`
	Role      string `json:"role"      yaml:"role"`
	User      string `json:"user"      yaml:"user"`
}

func newBindingResult(response *authorityv1.CreateBindingResponse) bindingResult {
	return bindingResult{
		ID:        response.GetId(),
		ProjectID: response.GetProjectId(),
		Role:      response.GetRoleName(),
		User:      response.GetUserName(),
	}
}

func (result bindingResult) String() string {
	return strings.Join(result.tableRow(), "\t")
}

func (result bindingResult) tableRow() []string {
	return []string{
		quoteIfUnprintable(result.Role),
		quoteIfUnprintable(result.User),
		quoteIfUnprintable(result.ID),
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
			Role:      binding.GetRoleName(),
			User:      binding.GetUserName(),
		})
	}

	return result
}

func (result bindingsResult) String() string {
	if len(result.Bindings) == 0 {
		return "No bindings in the project. Run `tidegate binding create` to give a role to a user."
	}

	rows := make([][]string, 0, len(result.Bindings))
	for _, binding := range result.Bindings {
		rows = append(rows, binding.tableRow())
	}

	return tableText([]string{"ROLE", "USER", "ID"}, rows)
}

func bindingCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "binding",
		Usage: "give roles of a project to users and list who has which role",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "give a role of a project to a user and print the role, the user and the binding's ID",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "project",
						Usage: "name of the project that gets the binding",
					},
					&cli.StringFlag{
						Name:  "role",
						Usage: "name of the role of the project that the user gets",
					},
					&cli.StringFlag{
						Name:  "user",
						Usage: "name of the user who gets the role",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return bindingCreate(ctx, cmd, store)
				},
			},
			{
				Name:  "list",
				Usage: "print the role, the user and the ID of every binding of a project",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "project",
						Usage: "name of the project whose bindings are listed",
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

	if cmd.String("project") == "" {
		return errors.New("want a project, name it with --project")
	}

	if cmd.String("role") == "" {
		return errors.New("want a role, name it with --role")
	}

	if cmd.String("user") == "" {
		return errors.New("want a user, name them with --user")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Binding().CreateBinding(ctx, connect.NewRequest(&authorityv1.CreateBindingRequest{
		ProjectName: cmd.String("project"),
		RoleName:    cmd.String("role"),
		UserName:    cmd.String("user"),
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

	if cmd.String("project") == "" {
		return errors.New("want a project, name it with --project")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Binding().ListBindings(ctx, connect.NewRequest(&authorityv1.ListBindingsRequest{
		ProjectName: cmd.String("project"),
	}))
	if err != nil {
		return callFailed("listing the bindings", connection, err)
	}

	return printResult(cmd, newBindingsResult(response.Msg.GetBindings()))
}
