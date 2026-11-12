package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionsResultPrintsATableSortedByName(t *testing.T) {
	t.Parallel()

	// arrange
	result := newConnectionsResult(threeConnections())

	// act
	text := result.String()

	// assert
	assert.Equal(t, ""+
		"NAME        SERVER                             TOKEN\n"+
		"admin       https://tidegate.example           saved\n"+
		"production  https://tidegate.example           saved\n"+
		"staging     https://staging.example/tidegate/  saved", text)
}

func TestConnectionsResultNeverContainsAToken(t *testing.T) {
	t.Parallel()

	// act
	result := newConnectionsResult(threeConnections())

	// assert
	assert.NotContains(t, result.String(), "production token")
}

func TestConnectionsResultSaysWhenNothingIsSaved(t *testing.T) {
	t.Parallel()

	// act
	text := newConnectionsResult(nil).String()

	// assert
	assert.Contains(t, text, "tidegate login <url>")
}

func TestConnectionResultMarksAConnectionWithoutAToken(t *testing.T) {
	t.Parallel()

	// act
	text := newConnectionResult("production", connectionWithoutToken()).String()

	// assert
	assert.Equal(t, "production\thttps://tidegate.example\tnone", text)
}

func TestConnectionResultQuotesANameThatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// act
	text := newConnectionResult("a\x1b[31m", connectionWithoutToken()).String()

	// assert
	assert.NotContains(t, text, "\x1b")
}
