//go:build unix

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

func TestLoadRefusesAFileThatOthersCanUse(t *testing.T) {
	t.Parallel()

	for name, mode := range map[string]os.FileMode{
		"readable by the group":  0o640,
		"writable by the group":  0o620,
		"readable by others":     0o604,
		"writable by others":     0o602,
		"executable by everyone": 0o711,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			path := savedIn(t, privateDirectory(t))
			require.NoError(t, os.Chmod(path, mode))

			// act
			_, err := connections.Load(path)

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "chmod 600 "+path)
		})
	}
}

func TestLoadRefusesADirectoryThatOthersCanUse(t *testing.T) {
	t.Parallel()

	for name, mode := range map[string]os.FileMode{
		"usable by the group":  0o750,
		"writable by others":   0o702,
		"listable by everyone": 0o755,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			directory := privateDirectory(t)
			path := savedIn(t, directory)
			require.NoError(t, os.Chmod(directory, mode))

			// act
			_, err := connections.Load(path)

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "chmod 700 "+directory)
		})
	}
}

func TestLoadRefusesAFileThatIsASymbolicLink(t *testing.T) {
	t.Parallel()

	// arrange
	elsewhere := savedIn(t, privateDirectory(t))
	path := filepath.Join(privateDirectory(t), "connections.json")
	require.NoError(t, os.Symlink(elsewhere, path))

	// act
	_, err := connections.Load(path)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbolic link")
}

func TestLoadRefusesADirectoryThatIsASymbolicLink(t *testing.T) {
	t.Parallel()

	// arrange
	elsewhere := privateDirectory(t)
	savedIn(t, elsewhere)

	directory := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(elsewhere, directory))

	// act
	_, err := connections.Load(filepath.Join(directory, "connections.json"))

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbolic link")
}

func TestLoadReturnsNoConnectionsWhenTheDirectoryIsMissing(t *testing.T) {
	t.Parallel()

	// act
	loaded, err := connections.Load(filepath.Join(t.TempDir(), "tidegate", "connections.json"))

	// assert
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestSignInRefusesADirectoryThatOthersCanUseAndWritesNothing(t *testing.T) {
	t.Parallel()

	// arrange
	directory := privateDirectory(t)
	require.NoError(t, os.Chmod(directory, 0o755)) //nolint:gosec // The test needs a directory that is open to others.

	// act
	_, err := connections.SignIn(
		filepath.Join(directory, "connections.json"),
		"production",
		connections.Connection{Server: server, Token: secret.New(token)},
	)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chmod 700 "+directory)

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDefaultPathRefusesAConfigurationDirectoryThatIsNotAbsolute(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "relative")

	// act
	_, err := connections.DefaultPath()

	// assert
	require.Error(t, err)
	assert.NotErrorIs(t, err, connections.ErrNoConfigDir)
	assert.Contains(t, err.Error(), "absolute")
}

func TestDefaultPathSaysWhenThereIsNoConfigurationDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	// act
	_, err := connections.DefaultPath()

	// assert
	require.ErrorIs(t, err, connections.ErrNoConfigDir)
}
