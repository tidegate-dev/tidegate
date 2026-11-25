// Command phemera is the command line interface for a phemera installation.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/phemera-dev/phemera/cli/internal/command"
	"github.com/phemera-dev/phemera/cli/internal/connections"
)

func main() {
	root := command.New(connections.NewStore())

	err := root.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", command.Message(err))
		os.Exit(1)
	}
}
