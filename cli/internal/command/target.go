package command

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func connectToTarget(cmd *cli.Command) (*authority.Clients, target, error) {
	chosen, err := pickTargetForCommand(cmd)
	if err != nil {
		return nil, target{}, err
	}

	clients, err := authority.Connect(chosen.server, chosen.token)
	if err != nil {
		return nil, target{}, err
	}

	return clients, chosen, nil
}

func pickTargetForCommand(cmd *cli.Command) (target, error) {
	options := readTargetOptions(cmd)

	var byName map[string]connections.Connection

	if options.usesConnectionsFile() {
		var err error

		byName, err = loadConnections()
		if err != nil {
			return target{}, err
		}
	}

	return pickTarget(options, byName)
}

func readTargetOptions(cmd *cli.Command) targetOptions {
	return targetOptions{
		connection:            cmd.String("connection"),
		environmentConnection: os.Getenv("TIDEGATE_CONNECTION"),
		server:                os.Getenv("TIDEGATE_SERVER"),
		token:                 secret.New(os.Getenv("TIDEGATE_TOKEN")),
	}
}

func loadConnections() (map[string]connections.Connection, error) {
	path, err := connections.DefaultPath()
	// A container may have no configuration directory.
	if errors.Is(err, connections.ErrNoConfigDir) {
		return map[string]connections.Connection{}, nil
	}

	if err != nil {
		return nil, err
	}

	return connections.Load(path)
}

// targetOptions holds --connection, TIDEGATE_CONNECTION, TIDEGATE_SERVER and TIDEGATE_TOKEN as the user gave them.
type targetOptions struct {
	connection            string
	environmentConnection string
	server                string
	token                 secret.String
}

func (s targetOptions) connectionName() string {
	return cmp.Or(s.connection, s.environmentConnection)
}

func (s targetOptions) usesConnectionsFile() bool {
	return s.server == "" && s.token.Reveal() == ""
}

type target struct {
	connection      string
	server          string
	token           secret.String
	fromEnvironment bool
}

func (t target) newCallError(action string, err error) error {
	return &callError{action: action, target: t, err: err}
}

func pickTarget(options targetOptions, byName map[string]connections.Connection) (target, error) {
	switch {
	case options.usesConnectionsFile():
		return pickConnection(options.connectionName(), byName)
	case options.connection != "":
		return target{}, errors.New("give --connection, or TIDEGATE_SERVER and TIDEGATE_TOKEN, not both")
	case options.server == "":
		return target{}, errors.New("TIDEGATE_TOKEN is set, but TIDEGATE_SERVER isn't. Set both, or unset TIDEGATE_TOKEN to use a saved connection")
	default:
		return target{server: options.server, token: options.token, fromEnvironment: true}, nil
	}
}

func pickConnection(name string, byName map[string]connections.Connection) (target, error) {
	if name != "" {
		return findConnection(name, byName)
	}

	switch len(byName) {
	case 0:
		return target{}, errors.New("no saved connections. Run `tidegate login <url>` to save one")
	case 1:
		for name, connection := range byName {
			return targetFor(name, connection), nil
		}
	}

	return target{}, fmt.Errorf("several saved connections: %s. Choose one with --connection or TIDEGATE_CONNECTION", quotedNames(byName))
}

func findConnection(name string, byName map[string]connections.Connection) (target, error) {
	connection, ok := byName[name]
	if !ok && len(byName) == 0 {
		return target{}, fmt.Errorf("%w. To save it, run `tidegate login <url> --connection %q`", connectionNotFound(name, byName), name)
	}

	if !ok {
		return target{}, connectionNotFound(name, byName)
	}

	return targetFor(name, connection), nil
}

func connectionNotFound(name string, byName map[string]connections.Connection) error {
	if len(byName) == 0 {
		return fmt.Errorf("no saved connection named %q", name)
	}

	return fmt.Errorf("no saved connection named %q. Saved connections: %s", name, quotedNames(byName))
}

func targetFor(name string, connection connections.Connection) target {
	return target{connection: name, server: connection.Server, token: connection.Token}
}

func quotedNames(byName map[string]connections.Connection) string {
	quoted := make([]string, 0, len(byName))
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		quoted = append(quoted, strconv.Quote(name))
	}

	return strings.Join(quoted, ", ")
}
