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

type projectResult struct {
	ID   string `json:"id"   yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

func newProjectResult(project *authorityv1.Project) projectResult {
	return projectResult{ID: project.GetId(), Name: project.GetName()}
}

func (result projectResult) String() string {
	return strings.Join(result.tableRow(), "\t")
}

func (result projectResult) tableRow() []string {
	return []string{quoteIfUnprintable(result.Name), quoteIfUnprintable(result.ID)}
}

type projectsResult struct {
	Projects []projectResult `json:"projects" yaml:"projects"`
}

func newProjectsResult(projects []*authorityv1.Project) projectsResult {
	result := projectsResult{Projects: make([]projectResult, 0, len(projects))}
	for _, project := range projects {
		result.Projects = append(result.Projects, newProjectResult(project))
	}

	return result
}

func (result projectsResult) String() string {
	if len(result.Projects) == 0 {
		return "No projects. Run `tidegate project create <name>` to create one."
	}

	rows := make([][]string, 0, len(result.Projects))
	for _, project := range result.Projects {
		rows = append(rows, project.tableRow())
	}

	return tableText([]string{"NAME", "ID"}, rows)
}

func projectCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "project",
		Usage: "create projects and list them",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "create a project and print its name and ID",
				ArgsUsage: "<name>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return projectCreate(ctx, cmd, store)
				},
			},
			{
				Name:  "list",
				Usage: "print the name and the ID of every project",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return projectList(ctx, cmd, store)
				},
			},
		},
	}
}

func projectCreate(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one name")
	}

	name := cmd.Args().First()

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Project().CreateProject(ctx, connect.NewRequest(&authorityv1.CreateProjectRequest{Name: name}))
	if err != nil {
		return callFailed("creating the project", connection, err)
	}

	return printResult(cmd, newProjectResult(response.Msg.GetProject()))
}

func projectList(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 0 {
		return errors.New("want no arguments")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Project().ListProjects(ctx, connect.NewRequest(&authorityv1.ListProjectsRequest{}))
	if err != nil {
		return callFailed("listing the projects", connection, err)
	}

	return printResult(cmd, newProjectsResult(response.Msg.GetProjects()))
}
