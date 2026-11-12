package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

const (
	productionServer = "https://tidegate.example"
	stagingServer    = "https://staging.example/tidegate/"
	otherServer      = "https://other.example"
)

func threeConnections() map[string]connections.Connection {
	return map[string]connections.Connection{
		"production": {Server: productionServer, Token: secret.New("production token")},
		"staging":    {Server: stagingServer, Token: secret.New("staging token")},
		"admin":      {Server: productionServer, Token: secret.New("admin token")},
	}
}

func onlyProduction() map[string]connections.Connection {
	return map[string]connections.Connection{
		"production": {Server: productionServer, Token: secret.New("production token")},
	}
}

func connectionWithoutToken() connections.Connection {
	return connections.Connection{Server: productionServer}
}

func TestPickTargetTakesTheConnectionNamedByTheFlag(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickTarget(targetOptions{connection: "admin", environmentConnection: "staging"}, threeConnections())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "admin", chosen.connection)
	assert.Equal(t, productionServer, chosen.server)
	assert.Equal(t, "admin token", chosen.token.Reveal())
}

func TestPickTargetTakesTheConnectionNamedByTheEnvironment(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickTarget(targetOptions{environmentConnection: "staging"}, threeConnections())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", chosen.connection)
	assert.Equal(t, stagingServer, chosen.server)
	assert.Equal(t, "staging token", chosen.token.Reveal())
}

func TestPickTargetTakesTheOnlySavedConnectionWhenNoneIsNamed(t *testing.T) {
	t.Parallel()

	// act
	chosen, err := pickTarget(targetOptions{}, onlyProduction())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", chosen.connection)
	assert.Equal(t, productionServer, chosen.server)
	assert.Equal(t, "production token", chosen.token.Reveal())
}

func TestPickTargetTakesTheServerOfAConnectionThatIsSignedOut(t *testing.T) {
	t.Parallel()

	// arrange
	byName := map[string]connections.Connection{"production": connectionWithoutToken()}

	// act
	chosen, err := pickTarget(targetOptions{}, byName)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", chosen.connection)
	assert.Equal(t, productionServer, chosen.server)
	assert.Empty(t, chosen.token.Reveal())
}

func TestConnectionNotFoundDoesNotSayHowToSaveTheName(t *testing.T) {
	t.Parallel()

	// act
	err := connectionNotFound("testing", nil)

	// assert
	assert.NotContains(t, err.Error(), "tidegate login")
}

func TestPickTargetFailsForANameThatIsNotSaved(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickTarget(targetOptions{connection: "testing"}, threeConnections())

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no saved connection named "testing"`)
	assert.Contains(t, err.Error(), `"admin", "production", "staging"`)
}

func TestPickTargetSaysHowToSaveANameWhenNothingIsSaved(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickTarget(targetOptions{connection: "testing"}, nil)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `tidegate login <url> --connection "testing"`)
}

func TestPickTargetFailsWhenSeveralConnectionsAreSavedAndNoneIsNamed(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickTarget(targetOptions{}, threeConnections())

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"admin", "production", "staging"`)
	assert.Contains(t, err.Error(), "--connection")
	assert.Contains(t, err.Error(), "TIDEGATE_CONNECTION")
}

func TestPickTargetSaysToRunLoginWhenNothingIsSaved(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickTarget(targetOptions{}, nil)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Run `tidegate login <url>`")
}

func TestPickTargetQuotesSavedNames(t *testing.T) {
	t.Parallel()

	// arrange
	byName := map[string]connections.Connection{
		"a\x1b[31m": {Server: productionServer},
		"b":         {Server: productionServer},
	}

	// act
	_, err := pickTarget(targetOptions{}, byName)

	// assert
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "\x1b")
}

func TestPickTargetTakesTheServerAndTheTokenFromTheEnvironmentAndNoSavedConnection(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		options targetOptions
		token   string
	}{
		{
			name:    "a server and a token",
			options: targetOptions{server: otherServer, token: secret.New("token from the environment")},
			token:   "token from the environment",
		},
		{
			name:    "only a server",
			options: targetOptions{server: otherServer},
		},
		{
			name: "a server, a token and a connection from the environment",
			options: targetOptions{
				server:                otherServer,
				token:                 secret.New("token from the environment"),
				environmentConnection: "production",
			},
			token: "token from the environment",
		},
		{
			name:    "the server of a saved connection",
			options: targetOptions{server: productionServer},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// act
			chosen, err := pickTarget(test.options, threeConnections())

			// assert
			require.NoError(t, err)
			assert.Empty(t, chosen.connection)
			assert.Equal(t, test.options.server, chosen.server)
			assert.Equal(t, test.token, chosen.token.Reveal())
			assert.True(t, chosen.fromEnvironment)
		})
	}
}

func TestPickTargetFailsForATokenFromTheEnvironmentWithoutAServer(t *testing.T) {
	t.Parallel()

	// act
	_, err := pickTarget(targetOptions{token: secret.New("token from the environment")}, onlyProduction())

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TIDEGATE_SERVER")
}

func TestPickTargetFailsForTheFlagNextToAServerOrATokenFromTheEnvironment(t *testing.T) {
	t.Parallel()

	for name, options := range map[string]targetOptions{
		"a server": {connection: "admin", server: otherServer},
		"a token":  {connection: "admin", token: secret.New("token from the environment")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := pickTarget(options, threeConnections())

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not both")
		})
	}
}

func TestUsesConnectionsFileIsTrueUnlessAServerOrATokenIsInTheEnvironment(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		options targetOptions
		needed  bool
	}{
		{name: "nothing given", options: targetOptions{}, needed: true},
		{name: "a name", options: targetOptions{connection: "admin"}, needed: true},
		{name: "a name from the environment", options: targetOptions{environmentConnection: "admin"}, needed: true},
		{name: "a server", options: targetOptions{server: otherServer}, needed: false},
		{name: "a token", options: targetOptions{token: secret.New("token")}, needed: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// act
			needed := test.options.usesConnectionsFile()

			// assert
			assert.Equal(t, test.needed, needed)
		})
	}
}
