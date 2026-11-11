package command

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCredentialsReadsTwoLinesWithoutATerminal(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		input string
	}{
		{name: "LF line endings", input: "alice\nhunter2\n"},
		{name: "CRLF line endings", input: "alice\r\nhunter2\r\n"},
		{name: "no line ending after the password", input: "alice\nhunter2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// act
			userName, password, err := readCredentials(t.Context(), strings.NewReader(test.input), io.Discard)

			// assert
			require.NoError(t, err)
			assert.Equal(t, "alice", userName)
			assert.Equal(t, "hunter2", password)
		})
	}
}

func TestReadCredentialsFailsWhenTheInputEndsEarly(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		input   string
		message string
	}{
		{name: "no input", input: "", message: "no user name"},
		{name: "only a user name", input: "alice\n", message: "no password"},
		{name: "only a user name without a line ending", input: "alice", message: "no password"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// act
			_, _, err := readCredentials(t.Context(), strings.NewReader(test.input), io.Discard)

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.message)
		})
	}
}
