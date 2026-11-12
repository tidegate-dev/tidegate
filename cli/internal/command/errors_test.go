package command

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

func unauthenticated(chosen target) error {
	return chosen.newCallError("storing the text", connect.NewError(connect.CodeUnauthenticated, errors.New("no session")))
}

func TestMessageSaysToSignInToTheConnectionThatHasNoSession(t *testing.T) {
	t.Parallel()

	// act
	message := Message(unauthenticated(target{connection: "production", server: productionServer}))

	// assert
	assert.Contains(t, message, "Run `tidegate login --connection \"production\"`")
	assert.NotContains(t, message, "TIDEGATE_TOKEN")
}

func TestMessageSaysToGetANewTokenWhenTheAuthorityRefusedTheOneInTheEnvironment(t *testing.T) {
	t.Parallel()

	// arrange
	chosen := target{server: productionServer, token: secret.New("token from the environment"), fromEnvironment: true}

	// act
	message := Message(unauthenticated(chosen))

	// assert
	assert.Contains(t, message, "TIDEGATE_TOKEN")
	assert.Contains(t, message, "tidegate login <url> --print-token")
	assert.NotContains(t, message, "token from the environment")
}

func TestMessageSaysToSetATokenWhenOnlyTheServerIsInTheEnvironment(t *testing.T) {
	t.Parallel()

	// act
	message := Message(unauthenticated(target{server: productionServer, fromEnvironment: true}))

	// assert
	assert.Contains(t, message, "Set TIDEGATE_TOKEN")
	assert.Contains(t, message, "unset TIDEGATE_SERVER")
}

func TestMessageNamesTheAuthorityThatCannotBeReached(t *testing.T) {
	t.Parallel()

	// arrange
	err := target{server: productionServer}.
		newCallError("pinging the authority", connect.NewError(connect.CodeUnavailable, errors.New("connection refused")))

	// act
	message := Message(err)

	// assert
	assert.Equal(t, "can't reach the authority at "+productionServer+": connection refused", message)
}

func TestMessageEndsWithTheHintOfTheError(t *testing.T) {
	t.Parallel()

	// arrange
	err := &callError{
		action: "signing out",
		target: target{server: productionServer},
		err:    connect.NewError(connect.CodeUnavailable, errors.New("connection refused")),
		hint:   "Try again later.",
	}

	// act
	message := Message(err)

	// assert
	assert.Equal(t, "can't reach the authority at "+productionServer+": connection refused. Try again later.", message)
}

func TestMessageQuotesAServerThatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// arrange
	err := target{server: "https://a.example\u202egro.live"}.
		newCallError("pinging the authority", connect.NewError(connect.CodeUnavailable, errors.New("connection refused")))

	// act
	message := Message(err)

	// assert
	assert.NotContains(t, message, "\u202e")
}
