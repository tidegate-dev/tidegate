package command

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

type connectionResult struct {
	Name       string `json:"name"       yaml:"name"`
	Server     string `json:"server"     yaml:"server"`
	TokenSaved bool   `json:"tokenSaved" yaml:"tokenSaved"`
}

func newConnectionResult(connection connections.Connection) connectionResult {
	return connectionResult{
		Name:       connection.Name,
		Server:     connection.Server,
		TokenSaved: connection.Token.Reveal() != "",
	}
}

func (result connectionResult) String() string {
	return strings.Join(result.tableRow(), "\t")
}

func (result connectionResult) tableRow() []string {
	tokenColumn := "none"
	if result.TokenSaved {
		tokenColumn = "saved"
	}

	return []string{quoteIfUnprintable(result.Name), quoteIfUnprintable(result.Server), tokenColumn}
}

type connectionsResult struct {
	Connections []connectionResult `json:"connections" yaml:"connections"`
}

func newConnectionsResult(saved []connections.Connection) connectionsResult {
	result := connectionsResult{Connections: make([]connectionResult, 0, len(saved))}
	for _, connection := range saved {
		result.Connections = append(result.Connections, newConnectionResult(connection))
	}

	return result
}

func (result connectionsResult) String() string {
	if len(result.Connections) == 0 {
		return "No saved connections. Run `tidegate login <url>` to save one."
	}

	var tableText strings.Builder

	table := tabwriter.NewWriter(&tableText, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(table, "NAME\tSERVER\tTOKEN")

	for _, connection := range result.Connections {
		_, _ = fmt.Fprintln(table, strings.Join(connection.tableRow(), "\t"))
	}

	_ = table.Flush()

	return strings.TrimSuffix(tableText.String(), "\n")
}

// quoteIfUnprintable returns text in quotes if a terminal would interpret a
// part of it, like an escape sequence.
func quoteIfUnprintable(text string) string {
	quoted := strconv.Quote(text)
	if quoted == `"`+text+`"` {
		return text
	}

	return quoted
}

func connectionsCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "connections",
		Usage: "list the saved connections, print the one in use, or remove one",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "print every saved connection, without its session token",
				Action: func(_ context.Context, cmd *cli.Command) error {
					return connectionsList(cmd, store)
				},
			},
			{
				Name:  "current",
				Usage: "print the saved connection that commands in this shell use",
				Action: func(_ context.Context, cmd *cli.Command) error {
					return connectionsCurrent(cmd, store)
				},
			},
			{
				Name:      "remove",
				Usage:     "end the session of a saved connection and remove the connection",
				ArgsUsage: "<name>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return connectionsRemove(ctx, cmd, store)
				},
			},
		},
	}
}

func connectionsList(cmd *cli.Command, store connections.Store) error {
	saved, err := store.List()
	if err != nil {
		return err
	}

	return printResult(cmd, newConnectionsResult(saved))
}

func connectionsCurrent(cmd *cli.Command, store connections.Store) error {
	connection, err := store.Current(cmd.String("connection"))
	if err != nil {
		return err
	}

	// A connection from TIDEGATE_SERVER and TIDEGATE_TOKEN has no name and isn't saved.
	if connection.Name == "" {
		return errors.New("TIDEGATE_SERVER or TIDEGATE_TOKEN is set, so commands in this shell use no saved connection")
	}

	return printResult(cmd, newConnectionResult(connection))
}

func connectionsRemove(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one name")
	}

	name := cmd.Args().First()

	removed, err := store.Remove(name)
	if err != nil {
		return err
	}

	err = endSession(ctx, removed)
	if err != nil {
		printWarning(cmd, "removed %q, but its session is still valid at the authority: %s", name, Message(err))
	}

	return nil
}
