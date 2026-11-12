package connections_test

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

const (
	server = "https://tidegate.example"
	token  = "correct horse battery staple"
)

// privateDirectory returns a new directory that only the owner can use.
func privateDirectory(t *testing.T) string {
	t.Helper()

	directory := filepath.Join(t.TempDir(), "tidegate")
	require.NoError(t, os.Mkdir(directory, 0o700))

	return directory
}

// savedIn signs "production" in with token in a new file in directory.
func savedIn(t *testing.T, directory string) string {
	t.Helper()

	path := filepath.Join(directory, "connections.json")

	_, err := connections.SignIn(path, "production", connections.Connection{Server: server, Token: secret.New(token)})
	require.NoError(t, err)

	return path
}

func TestLoadReturnsWhatSignInSaved(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	_, err := connections.SignIn(path, "staging", connections.Connection{
		Server: "https://staging.example/tidegate/",
		Token:  secret.New("another token"),
	})
	require.NoError(t, err)

	// act
	loaded, err := connections.Load(path)

	// assert
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, server, loaded["production"].Server)
	assert.Equal(t, token, loaded["production"].Token.Reveal())
	assert.Equal(t, "https://staging.example/tidegate/", loaded["staging"].Server)
	assert.Equal(t, "another token", loaded["staging"].Token.Reveal())
}

func TestLoadReturnsNoConnectionsWhenTheFileIsMissing(t *testing.T) {
	t.Parallel()

	// act
	loaded, err := connections.Load(filepath.Join(privateDirectory(t), "connections.json"))

	// assert
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestLoadNamesTheFileThatIsNotValid(t *testing.T) {
	t.Parallel()

	// arrange
	path := filepath.Join(privateDirectory(t), "connections.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	// act
	_, err := connections.Load(path)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
}

func TestSignInCreatesTheDirectoryForTheOwnerOnly(t *testing.T) {
	t.Parallel()

	// arrange
	directory := filepath.Join(t.TempDir(), "tidegate")

	// act
	savedIn(t, directory)

	// assert
	info, err := os.Stat(directory)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestSignInWritesAFileForTheOwnerOnly(t *testing.T) {
	t.Parallel()

	// act
	path := savedIn(t, privateDirectory(t))

	// assert
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSignInLeavesOnlyTheFileAndItsLockBehind(t *testing.T) {
	t.Parallel()

	// arrange
	directory := privateDirectory(t)

	// act
	savedIn(t, directory)

	// assert
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "connections.json", entries[0].Name())
	assert.Equal(t, "connections.json.lock", entries[1].Name())
}

func TestSignInKeepsTheOldFileWhenItCannotWriteTheNewOne(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root can write to a read-only directory")
	}

	// arrange
	directory := privateDirectory(t)
	path := savedIn(t, directory)
	require.NoError(t, os.Chmod(directory, 0o500))       //nolint:gosec // A directory needs the execute bit.
	t.Cleanup(func() { _ = os.Chmod(directory, 0o700) }) //nolint:gosec // A directory needs the execute bit.

	// act
	_, err := connections.SignIn(path, "production", connections.Connection{Server: server, Token: secret.New("new token")})

	// assert
	require.Error(t, err)

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Equal(t, token, loaded["production"].Token.Reveal())
}

func TestSignInReturnsTheConnectionItReplaces(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	replaced, err := connections.SignIn(path, "production", connections.Connection{Server: server, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)
	assert.Equal(t, server, replaced.Server)
	assert.Equal(t, token, replaced.Token.Reveal())

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "new token", loaded["production"].Token.Reveal())
}

func TestSignInReturnsNoConnectionForANewName(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	replaced, err := connections.SignIn(path, "staging", connections.Connection{Server: server, Token: secret.New("new token")})

	// assert
	require.NoError(t, err)
	assert.Empty(t, replaced.Server)
	assert.Empty(t, replaced.Token.Reveal())
}

func TestSignInKeepsEveryConnectionWhenManySignInAtOnce(t *testing.T) {
	t.Parallel()

	// arrange
	const logins = 50

	path := filepath.Join(t.TempDir(), "tidegate", "connections.json")

	var (
		group  sync.WaitGroup
		failed = make(chan error, logins)
	)

	// act
	for number := range logins {
		group.Go(func() {
			_, err := connections.SignIn(path, strconv.Itoa(number), connections.Connection{Server: server})
			failed <- err
		})
	}

	group.Wait()
	close(failed)

	// assert
	for err := range failed {
		require.NoError(t, err)
	}

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Len(t, loaded, logins)
}

func TestSignOutRemovesTheTokenAndKeepsTheConnection(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	err := connections.SignOut(path, "production", secret.New(token))

	// assert
	require.NoError(t, err)

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Equal(t, server, loaded["production"].Server)
	assert.Empty(t, loaded["production"].Token.Reveal())
}

func TestSignOutLeavesNoTokenFieldInTheFile(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	err := connections.SignOut(path, "production", secret.New(token))

	// assert
	require.NoError(t, err)

	content, err := os.ReadFile(path) //nolint:gosec // The test wrote the file itself.
	require.NoError(t, err)
	assert.NotContains(t, string(content), "token")
}

func TestSignOutKeepsATokenThatIsNotTheGivenOne(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	err := connections.SignOut(path, "production", secret.New("an older token"))

	// assert
	require.NoError(t, err)

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Equal(t, token, loaded["production"].Token.Reveal())
}

func TestSignOutSavesNothingForANameThatIsNotSaved(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	err := connections.SignOut(path, "staging", secret.New(token))

	// assert
	require.NoError(t, err)

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}

func TestRemoveDeletesTheConnectionAndReturnsIt(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	removed, found, err := connections.Remove(path, "production")

	// assert
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, server, removed.Server)
	assert.Equal(t, token, removed.Token.Reveal())

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestRemoveFindsNothingForANameThatIsNotSaved(t *testing.T) {
	t.Parallel()

	// arrange
	path := savedIn(t, privateDirectory(t))

	// act
	_, found, err := connections.Remove(path, "staging")

	// assert
	require.NoError(t, err)
	assert.False(t, found)

	loaded, err := connections.Load(path)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}
