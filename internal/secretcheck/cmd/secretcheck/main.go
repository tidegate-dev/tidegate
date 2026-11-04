// Command secretcheck runs the secretcheck analyzer, for `just lint`.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/tidegate-dev/tidegate/internal/secretcheck"
)

func main() {
	singlechecker.Main(secretcheck.Analyzer)
}
