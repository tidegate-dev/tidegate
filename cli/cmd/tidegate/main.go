// Command tidegate is the command line interface for a tidegate installation.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tidegate-dev/tidegate/cli/internal/command"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

func main() {
	root := command.New(connections.NewStore())

	err := root.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", command.Message(err))
		os.Exit(1)
	}
}
