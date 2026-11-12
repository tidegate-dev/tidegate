// Package connections keeps the authorities that the CLI is signed in to in a
// file. Each one is saved under a name, with the session token for it.
package connections

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/tidegate-dev/tidegate/internal/filelock"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// Connection is one authority and the session token for it.
type Connection struct {
	// Server is the URL of the authority, as the user gave it at login.
	Server string
	// Token is the session token that the authority at Server gave out. It is
	// empty after the user signed out.
	Token secret.String
}

type fileContent struct {
	Connections map[string]fileEntry `json:"connections"`
}

type fileEntry struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

const lockWaitLimit = 10 * time.Second

// ErrNoConfigDir says that the user has no configuration directory, like in a
// container that sets no HOME.
var ErrNoConfigDir = errors.New("no configuration directory")

// DefaultPath returns tidegate/connections.json in the user's configuration
// directory.
func DefaultPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding the directory for the saved connections: %w: %w", ErrNoConfigDir, err)
	}

	if !filepath.IsAbs(directory) {
		return "", fmt.Errorf("the directory for the saved connections, %q, isn't an absolute path. Set HOME or XDG_CONFIG_HOME to an absolute path", directory)
	}

	return filepath.Join(directory, "tidegate", "connections.json"), nil
}

// Load returns the connections in the file at path by name, and none if the
// file doesn't exist. It refuses a file or a directory that another user can
// use.
func Load(path string) (map[string]Connection, error) {
	err := errors.Join(checkOnlyOwnerHasAccess(filepath.Dir(path), true), checkOnlyOwnerHasAccess(path, false))
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path) //nolint:gosec // The path is the user's own configuration file.
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]Connection{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading the saved connections: %w", err)
	}

	var stored fileContent

	err = json.Unmarshal(content, &stored)
	if err != nil {
		return nil, fmt.Errorf("reading the saved connections in %s: %w", path, err)
	}

	connections := make(map[string]Connection, len(stored.Connections))
	for name, entry := range stored.Connections {
		connections[name] = Connection{Server: entry.Server, Token: secret.New(entry.Token)}
	}

	return connections, nil
}

// SignIn saves connection under name in the file at path, and returns the
// connection that was saved under that name before.
func SignIn(path, name string, connection Connection) (Connection, error) {
	var replaced Connection

	err := editUnderLock(path, func(connections map[string]Connection) bool {
		replaced = connections[name]
		connections[name] = connection

		return true
	})

	return replaced, err
}

// SignOut removes the token of the connection saved under name, if it is
// still token. The connection stays.
func SignOut(path, name string, token secret.String) error {
	return editUnderLock(path, func(connections map[string]Connection) bool {
		connection, ok := connections[name]
		if !ok || connection.Token.Reveal() != token.Reveal() {
			return false
		}

		connections[name] = Connection{Server: connection.Server}

		return true
	})
}

// Remove deletes the connection saved under name and returns it. It reports
// whether there was one.
func Remove(path, name string) (Connection, bool, error) {
	var (
		removed Connection
		found   bool
	)

	err := editUnderLock(path, func(connections map[string]Connection) bool {
		removed, found = connections[name]
		delete(connections, name)

		return found
	})

	return removed, found, err
}

// A reader sees the old file or the new one, never a part of it.
func save(path string, connections map[string]Connection) error {
	stored := fileContent{Connections: make(map[string]fileEntry, len(connections))}
	for name, connection := range connections {
		stored.Connections[name] = fileEntry{Server: connection.Server, Token: connection.Token.Reveal()}
	}

	content, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
	}

	directory := filepath.Dir(path)

	err = createPrivateDirectory(directory)
	if err != nil {
		return err
	}

	// os.CreateTemp creates the file with mode 0600.
	temporary, err := os.CreateTemp(directory, filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
	}

	err = writeAndClose(temporary, append(content, '\n'))
	if err == nil {
		err = os.Rename(temporary.Name(), path)
	}

	if err != nil {
		_ = os.Remove(temporary.Name())

		return fmt.Errorf("saving the connections: %w", err)
	}

	return nil
}

func createPrivateDirectory(directory string) error {
	err := os.MkdirAll(directory, 0o700)
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
	}

	return checkOnlyOwnerHasAccess(directory, true)
}

func writeAndClose(target *os.File, content []byte) error {
	_, err := target.Write(content)
	if err == nil {
		err = target.Sync()
	}

	return errors.Join(err, target.Close())
}

// editUnderLock loads the connections at path and saves them if edit reports
// that it changed them. Two tidegate processes that do this at once take
// turns, and one gives up after waiting 10 seconds for the other.
func editUnderLock(path string, edit func(connections map[string]Connection) bool) (err error) {
	err = createPrivateDirectory(filepath.Dir(path))
	if err != nil {
		return err
	}

	lock, err := filelock.Acquire(path+".lock", lockWaitLimit)
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
	}

	defer func() {
		err = errors.Join(err, lock.Release())
	}()

	connections, err := Load(path)
	if err != nil {
		return err
	}

	if !edit(connections) {
		return nil
	}

	return save(path, connections)
}
