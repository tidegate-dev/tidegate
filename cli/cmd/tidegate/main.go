// Command tidegate is the command-line client for the tidegate authority.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tidegate-dev/tidegate/cli/internal/command"
)

func main() {
	root := command.New()

	err := root.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", command.Message(root, err))
		os.Exit(1)
	}
}
