package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/terminal"
)

func TestReadPasswordReturnsMatchingEntries(t *testing.T) {
	t.Parallel()

	// arrange
	in := interactive("correct horse battery staple", "correct horse battery staple")

	// act
	password, err := readPassword(t.Context(), in)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "correct horse battery staple", password.Reveal())
}

func TestReadPasswordAcceptsEntriesThatDifferOnlyInTheirEncoding(t *testing.T) {
	t.Parallel()

	// arrange
	in := interactive("caf\u00e9 au lait, bitte", "cafe\u0301 au lait, bitte")

	// act
	_, err := readPassword(t.Context(), in)

	// assert
	require.NoError(t, err)
}

func TestReadPasswordRefusesDifferentEntries(t *testing.T) {
	t.Parallel()

	// arrange
	in := interactive("correct horse battery staple", "correct horse battery stapel")

	// act
	_, err := readPassword(t.Context(), in)

	// assert
	require.ErrorIs(t, err, errPasswordsDiffer)
}

func TestReadPasswordRefusesAnEmptyStdin(t *testing.T) {
	t.Parallel()

	// arrange
	in := terminal.New(strings.NewReader(""), io.Discard)

	// act
	_, err := readPassword(t.Context(), in)

	// assert
	require.EqualError(t, err, "no password on stdin")
}

// interactiveInput is a terminal where a person types entries, one for each
// prompt.
type interactiveInput struct {
	entries []string
}

func interactive(entries ...string) *interactiveInput {
	return &interactiveInput{entries: entries}
}

func (*interactiveInput) Interactive() bool {
	return true
}

func (*interactiveInput) ReadLine(context.Context) (string, error) {
	return "", io.EOF
}

func (i *interactiveInput) ReadSecret(context.Context, string) (string, error) {
	entry := i.entries[0]
	i.entries = i.entries[1:]

	return entry, nil
}
