// Command control-plane runs the tidegate control plane.
package main

import (
	"fmt"
	"os"

	"github.com/tidegate-dev/tidegate/control-plane/internal/config"
)

func main() {
	_, err := config.Load(os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Println("Hello from the tidegate control plane.")
}
