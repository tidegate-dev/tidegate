package command

import (
	"cmp"
	"errors"
	"fmt"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

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
