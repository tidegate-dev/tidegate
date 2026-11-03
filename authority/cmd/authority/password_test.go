package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmedPasswordReturnsMatchingEntries(t *testing.T) {
	t.Parallel()

	// arrange
	ask := answers("correct horse battery staple", "correct horse battery staple")

	// act
	password, err := confirmedPassword(ask)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "correct horse battery staple", password)
}

func TestConfirmedPasswordRefusesDifferentEntries(t *testing.T) {
	t.Parallel()

	// arrange
	ask := answers("correct horse battery staple", "correct horse battery stapel")

	// act
	_, err := confirmedPassword(ask)

	// assert
	require.ErrorIs(t, err, errPasswordsDiffer)
}

func TestPasswordLineIsTheFirstLineWithoutItsLineEnding(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]string{
		"LF":             "correct horse battery staple\nsecond line",
		"CRLF":           "correct horse battery staple\r\n",
		"no line ending": "correct horse battery staple",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			password, err := passwordLine(strings.NewReader(input))

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

func TestPasswordLineDoesntCutOffALongPassword(t *testing.T) {
	t.Parallel()

	// arrange
	input := strings.Repeat("a", 2000) + "\r\n"

	// act
	line, err := passwordLine(strings.NewReader(input))

	// assert
	require.NoError(t, err)
	assert.Greater(t, len(line), 1024, "a cut-off line would pass as a valid password")
}
