package proxy

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPreambleReturnsTheTicket(t *testing.T) {
	t.Parallel()

	// act
	ticket, _, err := readPreamble(strings.NewReader("phemera-session-v1 the-ticket\n"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, "the-ticket", ticket.Reveal())
}

func TestReadPreambleReturnsAReaderWithWhatFollowsTheLine(t *testing.T) {
	t.Parallel()

	// act
	_, reader, err := readPreamble(strings.NewReader("phemera-session-v1 the-ticket\nSSH-2.0-client\r\n"))

	// assert
	require.NoError(t, err)

	rest, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "SSH-2.0-client\r\n", string(rest))
}

func TestReadPreambleRefusesALineThatIsNoPreamble(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"nothing":                     "",
		"no line break":               "phemera-session-v1 the-ticket",
		"another version":             "phemera-session-v2 the-ticket\n",
		"no ticket":                   "phemera-session-v1\n",
		"empty ticket":                "phemera-session-v1 \n",
		"two tickets":                 "phemera-session-v1 one two\n",
		"carriage return":             "phemera-session-v1 the-ticket\r\n",
		"control character in ticket": "phemera-session-v1 the\x00ticket\n",
		"letter outside ASCII":        "phemera-session-v1 the-tickét\n",
		"an SSH client's first line":  "SSH-2.0-OpenSSH_10.0\r\n",
		"longer than the limit":       "phemera-session-v1 " + strings.Repeat("a", maxPreambleBytes) + "\n",
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, _, err := readPreamble(strings.NewReader(input))

			// assert
			require.ErrorIs(t, err, errInvalidPreamble)
		})
	}
}

func TestReadPreambleAcceptsALineOfExactlyTheLimit(t *testing.T) {
	t.Parallel()

	// arrange
	start := "phemera-session-v1 "
	ticket := strings.Repeat("a", maxPreambleBytes-len(start)-1)

	// act
	read, _, err := readPreamble(strings.NewReader(start + ticket + "\n"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, ticket, read.Reveal())
}

func FuzzReadPreambleTakesOnlyTheLineItAccepts(f *testing.F) {
	f.Add([]byte("phemera-session-v1 the-ticket\nSSH-2.0-client\r\n"))
	f.Add([]byte("phemera-session-v1 \n"))
	f.Add([]byte("SSH-2.0-OpenSSH_10.0\r\n"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, input []byte) {
		ticket, reader, err := readPreamble(bytes.NewReader(input))
		if err != nil {
			return
		}

		rest, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, string(input), "phemera-session-v1 "+ticket.Reveal()+"\n"+string(rest))
	})
}
