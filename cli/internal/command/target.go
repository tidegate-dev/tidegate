package command

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

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

// loginOptions holds the URL and the flags of tidegate login, and TIDEGATE_CONNECTION, as the user gave them.
type loginOptions struct {
	url                   string
	connection            string
	environmentConnection string
	printToken            bool
}

func pickLoginTarget(options loginOptions, byName map[string]connections.Connection) (target, error) {
	switch {
	case options.url == "" && options.printToken:
		return target{}, errors.New("--print-token needs a URL: tidegate login <url> --print-token")
	case options.url == "":
		return pickConnection(cmp.Or(options.connection, options.environmentConnection), byName)
	case options.printToken && options.connection != "":
		return target{}, errors.New("--print-token saves nothing, so it takes no --connection")
	case options.printToken:
		return target{server: options.url}, nil
	case options.connection != "":
		return checkNameIsFree(options.connection, options.url, byName)
	}

	matching := map[string]connections.Connection{}

	for name, connection := range byName {
		if connection.Server == options.url {
			matching[name] = connection
		}
	}

	switch len(matching) {
	case 0:
		return checkNameIsFree(options.url, options.url, byName)
	case 1:
		return pickConnection("", matching)
	default:
		return target{}, fmt.Errorf("several saved connections use %s: %s. Choose one with --connection", options.url, quotedNames(matching))
	}
}

func checkNameIsFree(name, url string, byName map[string]connections.Connection) (target, error) {
	existing, ok := byName[name]
	if ok && existing.Server != url {
		return target{}, fmt.Errorf(
			"the connection %q is saved for %s. Choose another name, or remove it first with `tidegate connections remove %q`",
			name, quoteIfUnprintable(existing.Server), name,
		)
	}

	return target{connection: name, server: url}, nil
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
