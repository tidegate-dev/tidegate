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

func TestCurrentRefusesAFileThatOtherUsersCanUse(t *testing.T) {
	for description, fileMode := range map[string]os.FileMode{
		"readable by the group":  0o640,
		"writable by the group":  0o620,
		"readable by others":     0o604,
		"writable by others":     0o602,
		"executable by everyone": 0o711,
	} {
		t.Run(description, func(t *testing.T) {
			// arrange
			store := newStore(t, onlyProductionFile)
			path := savedConnectionsPath(t)
			require.NoError(t, os.Chmod(path, fileMode))

			// act
			_, err := store.Current("")

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "chmod 600 "+path)
		})
	}
}

func TestCurrentRefusesADirectoryThatOtherUsersCanUse(t *testing.T) {
	for description, directoryMode := range map[string]os.FileMode{
		"usable by the group":  0o750,
		"writable by others":   0o702,
		"listable by everyone": 0o755,
	} {
		t.Run(description, func(t *testing.T) {
			// arrange
			store := newStore(t, onlyProductionFile)
			directory := filepath.Dir(savedConnectionsPath(t))
			require.NoError(t, os.Chmod(directory, directoryMode))

			// act
			_, err := store.Current("")

			// assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "chmod 700 "+directory)
		})
	}
}

func TestCurrentRefusesAFileThatIsASymbolicLink(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	realPath := filepath.Join(t.TempDir(), "connections.json")
	require.NoError(t, os.WriteFile(realPath, []byte(onlyProductionFile), 0o600))

	linkPath := savedConnectionsPath(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(linkPath), 0o700))
	require.NoError(t, os.Symlink(realPath, linkPath))

	// act
	_, err := store.Current("")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbolic link")
}

func TestCurrentRefusesADirectoryThatIsASymbolicLink(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	realDirectory := t.TempDir()
	require.NoError(t, os.Chmod(realDirectory, 0o700)) //nolint:gosec // A directory needs the execute bit.
	require.NoError(t, os.WriteFile(filepath.Join(realDirectory, "connections.json"), []byte(onlyProductionFile), 0o600))

	linkDirectory := filepath.Dir(savedConnectionsPath(t))
	require.NoError(t, os.MkdirAll(filepath.Dir(linkDirectory), 0o700))
	require.NoError(t, os.Symlink(realDirectory, linkDirectory))

	// act
	_, err := store.Current("")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbolic link")
}

func TestSaveRefusesADirectoryThatOtherUsersCanUseAndWritesNothing(t *testing.T) {
	// arrange
	store := newStore(t, noFile)

	directory := filepath.Dir(savedConnectionsPath(t))
	require.NoError(t, os.MkdirAll(directory, 0o700))
	require.NoError(t, os.Chmod(directory, 0o755)) //nolint:gosec // The test needs a directory that is open to others.

	// act
	_, err := store.Save(connections.Connection{Name: "new", Server: newServer, Token: secret.New("new token")})

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chmod 700 "+directory)

	filesInDirectory, err := os.ReadDir(directory)
	require.NoError(t, err)
	assert.Empty(t, filesInDirectory)
}

func TestCurrentRefusesAConfigurationDirectoryThatIsNotAnAbsolutePath(t *testing.T) {
	// arrange
	store := newStore(t, noFile)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "relative")

	// act
	_, err := store.Current("")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "absolute path")
}
