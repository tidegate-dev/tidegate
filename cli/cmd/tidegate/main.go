// Command tidegate is the command-line client for the tidegate control plane.
package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	err := newCommand().Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
