package connections_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

const (
	productionServer = "https://tidegate.example"
	stagingServer    = "https://staging.example/tidegate/"
	scriptServer     = "https://script.example"
	newServer        = "https://new.example"

	noFile = ""

	onlyProductionFile = `{
		"connections": {
			"production": {"server": "https://tidegate.example", "token": "production token"}
		}
	}`

	threeConnectionsFile = `{
		"connections": {
			"production": {"server": "https://tidegate.example", "token": "production token"},
			"staging": {"server": "https://staging.example/tidegate/", "token": "staging token"},
			"admin": {"server": "https://tidegate.example", "token": "admin token"}
		}
	}`
)

// savedConnectionsPath returns where the store keeps its file.
func savedConnectionsPath(t *testing.T) string {
	t.Helper()

	configDirectory, err := os.UserConfigDir()
	require.NoError(t, err)

	return filepath.Join(configDirectory, "tidegate", "connections.json")
}

// newStore returns a store whose file is in a new, empty configuration
// directory and holds fileText. With noFile, there is no file. The TIDEGATE_
// variables that the store reads are empty.
func newStore(t *testing.T, fileText string) connections.Store {
	t.Helper()

	configDirectory := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDirectory)
	// macOS takes the configuration directory from HOME.
	t.Setenv("HOME", configDirectory)
	t.Setenv("TIDEGATE_CONNECTION", "")
	t.Setenv("TIDEGATE_SERVER", "")
	t.Setenv("TIDEGATE_TOKEN", "")

	if fileText != noFile {
		path := savedConnectionsPath(t)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(fileText), 0o600))
	}

	return connections.NewStore()
}

func TestCurrentIsTheConnectionNamedByTheFlag(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	current, err := store.Current("admin")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "admin", current.Name)
	assert.Equal(t, productionServer, current.Server)
	assert.Equal(t, "admin token", current.Token.Reveal())
}

func TestCurrentIsTheConnectionNamedByTheVariableWhenThereIsNoFlag(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)
	t.Setenv("TIDEGATE_CONNECTION", "staging")

	// act
	current, err := store.Current("")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", current.Name)
	assert.Equal(t, stagingServer, current.Server)
	assert.Equal(t, "staging token", current.Token.Reveal())
}

func TestCurrentPrefersTheFlagOverTheVariable(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)
	t.Setenv("TIDEGATE_CONNECTION", "staging")

	// act
	current, err := store.Current("admin")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "admin", current.Name)
}

func TestCurrentIsTheOnlySavedConnectionWhenNoNameIsGiven(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)

	// act
	current, err := store.Current("")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", current.Name)
	assert.Equal(t, productionServer, current.Server)
	assert.Equal(t, "production token", current.Token.Reveal())
}

func TestCurrentHasNoTokenForAConnectionThatIsSignedOut(t *testing.T) {
	// arrange
	store := newStore(t, `{"connections": {"production": {"server": "https://tidegate.example"}}}`)

	// act
	current, err := store.Current("")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", current.Name)
	assert.Empty(t, current.Token.Reveal())
}

func TestCurrentFailsWhenSeveralAreSavedAndNoNameIsGiven(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.Current("")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"admin", "production", "staging"`)
	assert.Contains(t, err.Error(), "--connection")
	assert.Contains(t, err.Error(), "TIDEGATE_CONNECTION")
}

func TestCurrentSaysToRunLoginWhenNothingIsSaved(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	_, err := store.Current("")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Run `tidegate login <url>`")
}

func TestCurrentFailsForANameThatIsNotSavedAndListsTheSavedNames(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.Current("testing")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no saved connection named "testing"`)
	assert.Contains(t, err.Error(), `"admin", "production", "staging"`)
}

func TestCurrentSaysHowToSaveANameWhenNothingIsSaved(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	_, err := store.Current("testing")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no saved connection named "testing"`)
	assert.Contains(t, err.Error(), `tidegate login <url> --connection "testing"`)
}

func TestCurrentQuotesSavedNamesThatATerminalWouldInterpret(t *testing.T) {
	// arrange
	store := newStore(t, `{"connections": {"a\u001b[31m": {"server": "https://tidegate.example"}}}`)

	// act
	_, err := store.Current("testing")

	// assert
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "\x1b")
}

func TestCurrentNamesTheFileThatIsNotValid(t *testing.T) {
	// arrange
	store := newStore(t, "not json")

	// act
	_, err := store.Current("")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), savedConnectionsPath(t))
}

func TestCurrentIsMadeFromTheServerAndTheTokenVariables(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	t.Setenv("TIDEGATE_SERVER", scriptServer)
	t.Setenv("TIDEGATE_TOKEN", "script token")

	// act
	current, err := store.Current("")

	// assert
	require.NoError(t, err)
	assert.Empty(t, current.Name)
	assert.Equal(t, scriptServer, current.Server)
	assert.Equal(t, "script token", current.Token.Reveal())
}

func TestCurrentFromTheServerVariableAloneHasNoToken(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	t.Setenv("TIDEGATE_SERVER", productionServer)

	// act
	current, err := store.Current("")

	// assert
	require.NoError(t, err)
	assert.Empty(t, current.Name)
	assert.Equal(t, productionServer, current.Server)
	assert.Empty(t, current.Token.Reveal())
}

func TestCurrentFromTheVariablesIgnoresTheConnectionVariable(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)
	t.Setenv("TIDEGATE_CONNECTION", "staging")
	t.Setenv("TIDEGATE_SERVER", scriptServer)
	t.Setenv("TIDEGATE_TOKEN", "script token")

	// act
	current, err := store.Current("")

	// assert
	require.NoError(t, err)
	assert.Empty(t, current.Name)
	assert.Equal(t, scriptServer, current.Server)
}

func TestCurrentFromTheVariablesDoesNotReadTheFile(t *testing.T) {
	// arrange
	store := newStore(t, "not json")
	t.Setenv("TIDEGATE_SERVER", scriptServer)
	t.Setenv("TIDEGATE_TOKEN", "script token")

	// act
	current, err := store.Current("")

	// assert
	require.NoError(t, err)
	assert.Equal(t, scriptServer, current.Server)
}

func TestCurrentFailsForATokenVariableWithoutAServerVariable(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	t.Setenv("TIDEGATE_TOKEN", "script token")

	// act
	_, err := store.Current("")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TIDEGATE_SERVER isn't")
}

func TestCurrentFailsForTheFlagNextToTheServerOrTheTokenVariable(t *testing.T) {
	for _, variable := range []string{"TIDEGATE_SERVER", "TIDEGATE_TOKEN"} {
		t.Run(variable, func(t *testing.T) {
			// arrange
			store := newStore(t, threeConnectionsFile)
			t.Setenv(variable, "set")

			// act
			_, err := store.Current("admin")

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not both")
		})
	}
}

func TestForLoginWithAURLAndAFlagIsSavedUnderTheNameFromTheFlag(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	forLogin, err := store.ForLogin(newServer, "new")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "new", forLogin.Name)
	assert.Equal(t, newServer, forLogin.Server)
	assert.Empty(t, forLogin.Token.Reveal())
}

func TestForLoginWithAURLAndNoFlagIsSavedUnderTheURL(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	forLogin, err := store.ForLogin(newServer, "")

	// assert
	require.NoError(t, err)
	assert.Equal(t, newServer, forLogin.Name)
	assert.Equal(t, newServer, forLogin.Server)
}

func TestForLoginWithAURLDoesNotTakeTheNameFromTheConnectionVariable(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)
	t.Setenv("TIDEGATE_CONNECTION", "production")

	// act
	forLogin, err := store.ForLogin(newServer, "")

	// assert
	require.NoError(t, err)
	assert.Equal(t, newServer, forLogin.Name)
	assert.Equal(t, newServer, forLogin.Server)
}

func TestForLoginWithASavedURLAndANewFlagNameIsASecondConnectionToThatServer(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	forLogin, err := store.ForLogin(productionServer, "third")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "third", forLogin.Name)
	assert.Equal(t, productionServer, forLogin.Server)
}

func TestForLoginWithAFlagNameThatIsSavedForTheSameURLIsThatConnection(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	forLogin, err := store.ForLogin(stagingServer, "staging")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", forLogin.Name)
	assert.Equal(t, stagingServer, forLogin.Server)
	assert.Empty(t, forLogin.Token.Reveal())
}

func TestForLoginFailsForAFlagNameThatIsSavedForAnotherURL(t *testing.T) {
	for _, url := range []string{newServer, productionServer + "/"} {
		t.Run(url, func(t *testing.T) {
			// arrange
			store := newStore(t, threeConnectionsFile)

			// act
			_, err := store.ForLogin(url, "production")

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), productionServer)
			assert.Contains(t, err.Error(), "tidegate connections remove")
		})
	}
}

func TestForLoginQuotesASavedServerThatATerminalWouldInterpret(t *testing.T) {
	// arrange
	store := newStore(t, `{"connections": {"production": {"server": "https://a\u001b[31m.example"}}}`)

	// act
	_, err := store.ForLogin(newServer, "production")

	// assert
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "\x1b")
}

func TestForLoginWithAURLThatOneConnectionHasIsThatConnection(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	forLogin, err := store.ForLogin(stagingServer, "")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", forLogin.Name)
	assert.Equal(t, stagingServer, forLogin.Server)
}

func TestForLoginFailsForAURLThatSeveralConnectionsHave(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.ForLogin(productionServer, "")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"admin", "production"`)
	assert.NotContains(t, err.Error(), "staging")
	assert.Contains(t, err.Error(), "--connection")
}

func TestForLoginFailsForAURLThatIsTheNameOfAConnectionToAnotherServer(t *testing.T) {
	// arrange
	store := newStore(t, `{"connections": {"https://new.example": {"server": "https://tidegate.example"}}}`)

	// act
	_, err := store.ForLogin(newServer, "")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tidegate connections remove")
}

func TestForLoginWithoutAURLIsTheConnectionNamedByTheFlagWithoutItsToken(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)
	t.Setenv("TIDEGATE_CONNECTION", "admin")

	// act
	forLogin, err := store.ForLogin("", "staging")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", forLogin.Name)
	assert.Equal(t, stagingServer, forLogin.Server)
	assert.Empty(t, forLogin.Token.Reveal())
}

func TestForLoginWithoutAURLIsTheConnectionNamedByTheVariable(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)
	t.Setenv("TIDEGATE_CONNECTION", "staging")

	// act
	forLogin, err := store.ForLogin("", "")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", forLogin.Name)
	assert.Equal(t, stagingServer, forLogin.Server)
}

func TestForLoginWithoutAURLIsTheOnlySavedConnection(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)

	// act
	forLogin, err := store.ForLogin("", "")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", forLogin.Name)
	assert.Equal(t, productionServer, forLogin.Server)
	assert.Empty(t, forLogin.Token.Reveal())
}

func TestForLoginWithoutAURLDoesNotUseTheServerAndTheTokenVariables(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	t.Setenv("TIDEGATE_SERVER", scriptServer)
	t.Setenv("TIDEGATE_TOKEN", "script token")

	// act
	forLogin, err := store.ForLogin("", "")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", forLogin.Name)
	assert.Equal(t, productionServer, forLogin.Server)
}

func TestForLoginWithoutAURLFailsWhenSeveralAreSavedAndNoNameIsGiven(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.ForLogin("", "")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"admin", "production", "staging"`)
}

func TestForLoginWithoutAURLSaysToGiveOneWhenNothingIsSaved(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	_, err := store.ForLogin("", "")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tidegate login <url>")
}

func TestForLoginWithoutAURLFailsForANameThatIsNotSaved(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.ForLogin("", "testing")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no saved connection named "testing"`)
}

func TestForLoginNamesTheFileThatIsNotValid(t *testing.T) {
	// arrange
	store := newStore(t, "not json")

	// act
	_, err := store.ForLogin(newServer, "")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), savedConnectionsPath(t))
}

func TestSaveStoresANewConnectionThatCurrentThenReturns(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	_, err := store.Save(connections.Connection{Name: "new", Server: newServer, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)

	current, err := store.Current("new")
	require.NoError(t, err)
	assert.Equal(t, "new", current.Name)
	assert.Equal(t, newServer, current.Server)
	assert.Equal(t, "new token", current.Token.Reveal())
}

func TestSaveReturnsNoConnectionForANameThatWasNotSaved(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)

	// act
	replaced, err := store.Save(connections.Connection{Name: "new", Server: newServer, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)
	assert.Empty(t, replaced.Name)
	assert.Empty(t, replaced.Server)
	assert.Empty(t, replaced.Token.Reveal())
}

func TestSaveReturnsTheConnectionThatWasSavedUnderTheNameBefore(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)

	// act
	replaced, err := store.Save(connections.Connection{Name: "production", Server: productionServer, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)
	assert.Equal(t, "production", replaced.Name)
	assert.Equal(t, productionServer, replaced.Server)
	assert.Equal(t, "production token", replaced.Token.Reveal())
}

func TestSaveReplacesTheTokenOfASavedConnection(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)

	// act
	_, err := store.Save(connections.Connection{Name: "production", Server: productionServer, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)

	current, err := store.Current("production")
	require.NoError(t, err)
	assert.Equal(t, "new token", current.Token.Reveal())
}

func TestSaveKeepsTheOtherConnections(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.Save(connections.Connection{Name: "new", Server: newServer, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)

	staging, err := store.Current("staging")
	require.NoError(t, err)
	assert.Equal(t, stagingServer, staging.Server)
	assert.Equal(t, "staging token", staging.Token.Reveal())
}

func TestSaveWritesTheFileWithTheConnectionsByName(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)

	// act
	_, err := store.Save(connections.Connection{Name: "new", Server: newServer, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)

	fileText, err := os.ReadFile(savedConnectionsPath(t))
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"connections": {
			"new": {"server": "https://new.example", "token": "new token"},
			"production": {"server": "https://tidegate.example", "token": "production token"}
		}
	}`, string(fileText))
}

func TestSaveCreatesTheFileAndItsDirectoryForTheOwnerOnly(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	_, err := store.Save(connections.Connection{Name: "new", Server: newServer, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)

	fileInfo, err := os.Stat(savedConnectionsPath(t))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())

	directoryInfo, err := os.Stat(filepath.Dir(savedConnectionsPath(t)))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), directoryInfo.Mode().Perm())
}

func TestSignOutRemovesTheTokenAndKeepsTheConnection(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	production := connections.Connection{Name: "production", Server: productionServer, Token: secret.New("production token")}

	// act
	err := store.SignOut(production)

	// assert
	require.NoError(t, err)

	current, err := store.Current("production")
	require.NoError(t, err)
	assert.Equal(t, productionServer, current.Server)
	assert.Empty(t, current.Token.Reveal())
}

func TestSignOutLeavesNoTokenFieldInTheFile(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	production := connections.Connection{Name: "production", Server: productionServer, Token: secret.New("production token")}

	// act
	err := store.SignOut(production)

	// assert
	require.NoError(t, err)

	fileText, err := os.ReadFile(savedConnectionsPath(t))
	require.NoError(t, err)
	assert.JSONEq(t, `{"connections": {"production": {"server": "https://tidegate.example"}}}`, string(fileText))
}

func TestSignOutKeepsATokenThatIsNotTheGivenOne(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	signedInEarlier := connections.Connection{Name: "production", Server: productionServer, Token: secret.New("an older token")}

	// act
	err := store.SignOut(signedInEarlier)

	// assert
	require.NoError(t, err)

	current, err := store.Current("production")
	require.NoError(t, err)
	assert.Equal(t, "production token", current.Token.Reveal())
}

func TestSignOutKeepsTheTokensOfTheOtherConnections(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)
	production := connections.Connection{Name: "production", Server: productionServer, Token: secret.New("production token")}

	// act
	err := store.SignOut(production)

	// assert
	require.NoError(t, err)

	admin, err := store.Current("admin")
	require.NoError(t, err)
	assert.Equal(t, "admin token", admin.Token.Reveal())
}

func TestSignOutChangesNothingForAConnectionThatIsNotSaved(t *testing.T) {
	// arrange
	store := newStore(t, onlyProductionFile)
	notSaved := connections.Connection{Name: "staging", Server: stagingServer, Token: secret.New("production token")}

	// act
	err := store.SignOut(notSaved)

	// assert
	require.NoError(t, err)

	fileText, err := os.ReadFile(savedConnectionsPath(t))
	require.NoError(t, err)
	assert.JSONEq(t, onlyProductionFile, string(fileText))
}

func TestListReturnsAllConnectionsSortedByName(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	saved, err := store.List()

	// assert
	require.NoError(t, err)
	require.Len(t, saved, 3)
	assert.Equal(t, "admin", saved[0].Name)
	assert.Equal(t, productionServer, saved[0].Server)
	assert.Equal(t, "admin token", saved[0].Token.Reveal())
	assert.Equal(t, "production", saved[1].Name)
	assert.Equal(t, "staging", saved[2].Name)
	assert.Equal(t, stagingServer, saved[2].Server)
}

func TestListReturnsNoConnectionsWhenNothingIsSaved(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	saved, err := store.List()

	// assert
	require.NoError(t, err)
	assert.Empty(t, saved)
}

func TestListNamesTheFileThatIsNotValid(t *testing.T) {
	// arrange
	store := newStore(t, "not json")

	// act
	_, err := store.List()

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), savedConnectionsPath(t))
}

func TestRemoveReturnsTheConnectionItDeleted(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	removed, err := store.Remove("staging")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "staging", removed.Name)
	assert.Equal(t, stagingServer, removed.Server)
	assert.Equal(t, "staging token", removed.Token.Reveal())
}

func TestRemoveDeletesTheConnectionAndKeepsTheOthers(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.Remove("staging")

	// assert
	require.NoError(t, err)

	saved, err := store.List()
	require.NoError(t, err)
	require.Len(t, saved, 2)
	assert.Equal(t, "admin", saved[0].Name)
	assert.Equal(t, "production", saved[1].Name)
}

func TestRemoveFailsForANameThatIsNotSavedAndListsTheSavedNames(t *testing.T) {
	// arrange
	store := newStore(t, threeConnectionsFile)

	// act
	_, err := store.Remove("testing")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no saved connection named "testing"`)
	assert.Contains(t, err.Error(), `"admin", "production", "staging"`)
}

func TestRemoveDoesNotSayToRunLoginWhenNothingIsSaved(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	// act
	_, err := store.Remove("testing")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no saved connection named "testing"`)
	assert.NotContains(t, err.Error(), "tidegate login")
}
