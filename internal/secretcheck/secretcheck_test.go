package secretcheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/tidegate-dev/tidegate/internal/secretcheck"
)

func TestAnalyzerReportsExportedSecretsWithoutSecretString(t *testing.T) {
	t.Parallel()

	analysistest.Run(t, analysistest.TestData(), secretcheck.Analyzer, "example")
}
