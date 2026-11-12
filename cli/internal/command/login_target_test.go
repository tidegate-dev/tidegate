package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

func TestPickLoginTargetSavesANewServerUnderItsURL(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickLoginTarget(loginOptions{url: otherServer}, threeConnections())

	// assert
	require.NoError(t, err)
	assert.Equal(t, otherServer, chosen.connection)
	assert.Equal(t, otherServer, chosen.server)
}

func TestPickLoginTargetSavesANewServerUnderTheGivenName(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickLoginTarget(loginOptions{url: otherServer, connection: "other"}, threeConnections())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "other", chosen.connection)
	assert.Equal(t, otherServer, chosen.server)
}

func TestPickLoginTargetSavesASecondConnectionToAServerUnderTheGivenName(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickLoginTarget(loginOptions{url: productionServer, connection: "third"}, threeConnections())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "third", chosen.connection)
	assert.Equal(t, productionServer, chosen.server)
}

func TestPickLoginTargetIgnoresTheNameFromTheEnvironmentForAURL(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickLoginTarget(loginOptions{url: otherServer, environmentConnection: "production"}, threeConnections())

	// assert
	require.NoError(t, err)
	assert.Equal(t, otherServer, chosen.connection)
	assert.Equal(t, otherServer, chosen.server)
}

func TestPickLoginTargetTakesTheOnlyConnectionSavedForAURL(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickLoginTarget(loginOptions{url: stagingServer}, threeConnections())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", chosen.connection)
	assert.Equal(t, stagingServer, chosen.server)
}

func TestPickLoginTargetFailsWhenSeveralConnectionsAreSavedForAURL(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickLoginTarget(loginOptions{url: productionServer}, threeConnections())

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"admin", "production"`)
	assert.NotContains(t, err.Error(), "staging")
	assert.Contains(t, err.Error(), "--connection")
}

func TestPickLoginTargetFailsForANameThatIsSavedForAnotherServer(t *testing.T) {
	t.Parallel()

	for _, url := range []string{otherServer, productionServer + "/"} {
		t.Run(url, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := pickLoginTarget(loginOptions{url: url, connection: "production"}, threeConnections())

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), productionServer)
			assert.Contains(t, err.Error(), "tidegate connections remove")
		})
	}
}

func TestPickLoginTargetFailsForAURLThatIsTheNameOfAConnectionToAnotherServer(t *testing.T) {
	t.Parallel()

	// arrange
	byName := map[string]connections.Connection{otherServer: {Server: productionServer}}

	// act
	_, err := pickLoginTarget(loginOptions{url: otherServer}, byName)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tidegate connections remove")
}

func TestPickLoginTargetQuotesASavedServerThatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// arrange
	byName := map[string]connections.Connection{"production": {Server: "https://a\x1b[31m.example"}}

	// act
	_, err := pickLoginTarget(loginOptions{url: otherServer, connection: "production"}, byName)

	// assert
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "\x1b")
}

func TestPickLoginTargetWithoutAURLTakesTheNamedConnection(t *testing.T) {
	t.Parallel()

	for name, options := range map[string]loginOptions{
		"from the flag":        {connection: "staging", environmentConnection: "admin"},
		"from the environment": {environmentConnection: "staging"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			chosen, err := pickLoginTarget(options, threeConnections())

			// assert
			require.NoError(t, err)
			assert.Equal(t, "staging", chosen.connection)
			assert.Equal(t, stagingServer, chosen.server)
		})
	}
}

func TestPickLoginTargetWithoutAURLTakesTheOnlySavedConnection(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickLoginTarget(loginOptions{}, onlyProduction())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", chosen.connection)
	assert.Equal(t, productionServer, chosen.server)
}

func TestPickLoginTargetWithoutAURLFailsForANameThatIsNotSaved(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickLoginTarget(loginOptions{connection: "testing"}, threeConnections())

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no saved connection named "testing"`)
}

func TestPickLoginTargetWithoutAURLFailsWhenSeveralConnectionsAreSaved(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickLoginTarget(loginOptions{}, threeConnections())

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"admin", "production", "staging"`)
}

func TestPickLoginTargetThatPrintsTheTokenTakesTheURLWithoutLookingAtSavedConnections(t *testing.T) {
	t.Parallel()

	for _, url := range []string{otherServer, stagingServer, productionServer} {
		t.Run(url, func(t *testing.T) {
			t.Parallel()

			// act
			chosen, err := pickLoginTarget(loginOptions{url: url, printToken: true}, threeConnections())

			// assert
			require.NoError(t, err)
			assert.Equal(t, url, chosen.server)
		})
	}
}

func TestPickLoginTargetThatPrintsTheTokenFailsForAURLWithAName(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickLoginTarget(loginOptions{url: otherServer, connection: "other", printToken: true}, nil)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "takes no --connection")
}

func TestPickLoginTargetThatPrintsTheTokenFailsWithoutAURL(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickLoginTarget(loginOptions{printToken: true}, onlyProduction())

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tidegate login <url> --print-token")
}
