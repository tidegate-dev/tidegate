package main

import (
	"context"
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

func TestReadPasswordLineReturnsTheFirstLineWithoutItsLineEnding(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]string{
		"LF":             "correct horse battery staple\nsecond line",
		"CRLF":           "correct horse battery staple\r\n",
		"no line ending": "correct horse battery staple",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			password, err := readPasswordLine(t.Context(), strings.NewReader(input))

			// assert
			require.NoError(t, err)
			assert.Equal(t, "correct horse battery staple", password)
		})
	}
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

func TestReadPasswordLineDoesntCutOffALongPassword(t *testing.T) {
	t.Parallel()

	// arrange
	input := strings.Repeat("a", 2000) + "\r\n"

	// act
	line, err := readPasswordLine(t.Context(), strings.NewReader(input))

	// assert
	require.NoError(t, err)
	assert.Greater(t, len(line), 1024, "a cut-off line would pass as a valid password")
}

func TestReadPasswordLineStopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	// arrange
	stdin, writer := io.Pipe()

	t.Cleanup(func() {
		_ = writer.Close()
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// act
	_, err := readPasswordLine(ctx, stdin)

	// assert
	require.ErrorIs(t, err, context.Canceled)
}
