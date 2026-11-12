package main

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAskPasswordTwiceReturnsMatchingEntries(t *testing.T) {
	t.Parallel()

	// arrange
	ask := answers("correct horse battery staple", "correct horse battery staple")

	// act
	password, err := askPasswordTwice(ask)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "correct horse battery staple", password.Reveal())
}

func TestAskPasswordTwiceAcceptsEntriesThatDifferOnlyInTheirEncoding(t *testing.T) {
	t.Parallel()

	// arrange
	ask := answers("caf\u00e9 au lait, bitte", "cafe\u0301 au lait, bitte")

	// act
	_, err := askPasswordTwice(ask)

	// assert
	require.NoError(t, err)
}

func TestAskPasswordTwiceRefusesDifferentEntries(t *testing.T) {
	t.Parallel()

	// arrange
	ask := answers("correct horse battery staple", "correct horse battery stapel")

	// act
	_, err := askPasswordTwice(ask)

	// assert
	require.ErrorIs(t, err, errPasswordsDiffer)
}

func TestReadPasswordRefusesAnEmptyStdin(t *testing.T) {
	t.Parallel()

	// act
	_, err := readPassword(t.Context(), strings.NewReader(""), io.Discard)

	// assert
	require.EqualError(t, err, "no password on stdin")
}

// answers returns a function that answers each prompt with the next of
// entries.
func answers(entries ...string) func(string) (string, error) {
	return func(string) (string, error) {
		entry := entries[0]
		entries = entries[1:]

		return entry, nil
	}
}
