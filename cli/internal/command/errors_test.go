package command

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/phemera-dev/phemera/cli/internal/connections"
)

func TestMessageQuotesAnAuthoritysErrorTextThatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	for name, code := range map[string]connect.Code{
		"an error the authority answered":  connect.CodeAlreadyExists,
		"an authority that is unavailable": connect.CodeUnavailable,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			connection := connections.Connection{Name: "first", Server: "https://phemera.example"}
			err := callFailed("creating the project", connection, connect.NewError(code, errors.New("taken\x1b[2J")))

			// act
			message := Message(err)

			// assert
			assert.NotContains(t, message, "\x1b")
			assert.Contains(t, message, `taken\x1b[2J`)
		})
	}
}
