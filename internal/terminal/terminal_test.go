package terminal

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadLineReturnsTheFirstLineWithoutItsLineEnding(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]string{
		"LF":             "correct horse battery staple\nsecond line",
		"CRLF":           "correct horse battery staple\r\n",
		"no line ending": "correct horse battery staple",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			terminal := New(strings.NewReader(input), io.Discard)

			// act
			line, err := terminal.ReadLine(t.Context())

			// assert
			require.NoError(t, err)
			assert.Equal(t, "correct horse battery staple", line)
		})
	}
}

func TestReadLineDoesntCutOffALongLine(t *testing.T) {
	t.Parallel()

	// arrange
	terminal := New(strings.NewReader(strings.Repeat("a", 2000)+"\r\n"), io.Discard)

	// act
	line, err := terminal.ReadLine(t.Context())

	// assert
	require.NoError(t, err)
	assert.Len(t, line, 2000)
}

func TestReadLineStopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	// arrange
	stdin, writer := io.Pipe()

	t.Cleanup(func() {
		_ = writer.Close()
	})

	terminal := New(stdin, io.Discard)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// act
	_, err := terminal.ReadLine(ctx)

	// assert
	require.ErrorIs(t, err, context.Canceled)
}
