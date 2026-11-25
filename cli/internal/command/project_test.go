package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectsResultTextNamesTheCreateCommandWhenThereAreNoProjects(t *testing.T) {
	t.Parallel()

	// arrange
	result := projectsResult{}

	// act
	text := result.String()

	// assert
	assert.Contains(t, text, "phemera project create <name>")
}

func TestProjectsResultTextQuotesANameThatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// arrange
	result := projectsResult{Projects: []projectResult{{ID: "1", Name: "a\x1b[2Jb"}}}

	// act
	text := result.String()

	// assert
	assert.NotContains(t, text, "\x1b")
	assert.Contains(t, text, `"a\x1b[2Jb"`)
}
