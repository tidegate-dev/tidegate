package command

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/phemera-dev/phemera/cli/internal/connections"
	"github.com/phemera-dev/phemera/cli/internal/sshsession"
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

func TestExitStatusIsTheStatusOfTheShellThatConnectOpened(t *testing.T) {
	t.Parallel()

	// arrange
	err := connectFailed(&sshsession.ExitError{Status: 3})

	// act
	status := ExitStatus(err)

	// assert
	assert.Equal(t, 3, status)
}

func TestExitStatusIs255ForAnErrorOfConnectItself(t *testing.T) {
	t.Parallel()

	// arrange
	err := connectFailed(errors.New("the ssh-proxy can't be reached"))

	// act
	status := ExitStatus(err)

	// assert
	assert.Equal(t, 255, status)
}

func TestExitStatusIs255WhenTheShellsStatusDoesNotFitInAByte(t *testing.T) {
	t.Parallel()

	// arrange
	err := connectFailed(&sshsession.ExitError{Status: 256})

	// act
	status := ExitStatus(err)

	// assert
	assert.Equal(t, 255, status)
}

func TestExitStatusIs1ForAnErrorOfAnotherCommandThanConnect(t *testing.T) {
	t.Parallel()

	// arrange
	err := errors.New("the disk is full")

	// act
	status := ExitStatus(err)

	// assert
	assert.Equal(t, 1, status)
}

func TestMessageIsEmptyWhenTheShellEndedWithAFailure(t *testing.T) {
	t.Parallel()

	// arrange
	err := connectFailed(&sshsession.ExitError{Status: 3})

	// act
	message := Message(err)

	// assert
	assert.Empty(t, message)
}
