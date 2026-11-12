package command

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
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

func newConnectionResult(name string, connection connections.Connection) connectionResult {
	return connectionResult{Name: name, Server: connection.Server, TokenSaved: connection.Token.Reveal() != ""}
}

func (r connectionResult) String() string {
	return strings.Join(r.tableRow(), "\t")
}

func (r connectionResult) tableRow() []string {
	token := "none"
	if r.TokenSaved {
		token = "saved"
	}

	return []string{quoteIfUnprintable(r.Name), quoteIfUnprintable(r.Server), token}
}

func quoteIfUnprintable(text string) string {
	quoted := strconv.Quote(text)
	if quoted == `"`+text+`"` {
		return text
	}

	return quoted
}

type connectionsResult struct {
	Connections []connectionResult `json:"connections" yaml:"connections"`
}

func newConnectionsResult(byName map[string]connections.Connection) connectionsResult {
	result := connectionsResult{Connections: make([]connectionResult, 0, len(byName))}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		result.Connections = append(result.Connections, newConnectionResult(name, byName[name]))
	}

	return result
}

func (r connectionsResult) String() string {
	if len(r.Connections) == 0 {
		return "No saved connections. Run `tidegate login <url>` to save one."
	}

	var text strings.Builder

	table := tabwriter.NewWriter(&text, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(table, "NAME\tSERVER\tTOKEN")

	for _, connection := range r.Connections {
		_, _ = fmt.Fprintln(table, strings.Join(connection.tableRow(), "\t"))
	}

	_ = table.Flush()

	return strings.TrimSuffix(text.String(), "\n")
}

func connectionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "connections",
		Usage: "list the saved connections, print the one in use, or remove one",
		Commands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "print every saved connection, without its session token",
				Action: connectionsList,
			},
			{
				Name:   "current",
				Usage:  "print the saved connection that commands in this shell use",
				Action: connectionsCurrent,
			},
			{
				Name:      "remove",
				Usage:     "end the session of a saved connection and remove the connection",
				ArgsUsage: "<name>",
				Action:    connectionsRemove,
			},
		},
	}
}

func connectionsList(_ context.Context, cmd *cli.Command) error {
	byName, err := loadConnections()
	if err != nil {
		return err
	}

	return printResult(cmd, newConnectionsResult(byName))
}

func connectionsCurrent(_ context.Context, cmd *cli.Command) error {
	options := readTargetOptions(cmd)
	if !options.usesConnectionsFile() {
		return errors.New("TIDEGATE_SERVER or TIDEGATE_TOKEN is set, so commands in this shell use no saved connection")
	}

	byName, err := loadConnections()
	if err != nil {
		return err
	}

	chosen, err := pickConnection(options.connectionName(), byName)
	if err != nil {
		return err
	}

	return printResult(cmd, newConnectionResult(chosen.connection, byName[chosen.connection]))
}

func connectionsRemove(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one name")
	}

	name := cmd.Args().First()

	byName, err := loadConnections()
	if err != nil {
		return err
	}

	if _, ok := byName[name]; !ok {
		return connectionNotFound(name, byName)
	}

	path, err := connections.DefaultPath()
	if err != nil {
		return err
	}

	removed, found, err := connections.Remove(path, name)
	if err != nil {
		return err
	}

	if !found {
		return connectionNotFound(name, nil)
	}

	err = endSession(ctx, removed)
	if err != nil {
		printWarning(cmd, "removed %q, but its session is still valid at the authority: %s", name, Message(err))
	}

	return nil
}
