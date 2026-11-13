// Package connections remembers the authorities that the user signed in to.
package connections

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tidegate-dev/tidegate/internal/filelock"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// Connection is one authority that the user signed in to.
type Connection struct {
	// Name is what the user called the connection at login. Without a name
	// from the user, it is the URL.
	Name string
	// Server is the URL of the authority, as the user gave it at login.
	Server string
	// Token is the session token that the authority gave out. It is empty
	// after the user signed out.
	Token secret.String
}

// Store is everything the commands ask about connections and every change
// they make to them.
type Store interface {
	// Current returns the connection that a command uses. connectionFlag is
	// the value of --connection. Without it, Current takes the name from
	// TIDEGATE_CONNECTION, and without that the only saved connection.
	//
	// If TIDEGATE_SERVER or TIDEGATE_TOKEN is set, Current returns a
	// connection without a name that is made from these two, and uses
	// nothing that is saved.
	Current(connectionFlag string) (Connection, error)

	// ForLogin returns the connection that tidegate login signs in to: its
	// server, the name it will be saved under, and no token. url is the
	// server the user gave, and is empty to sign in again to a saved
	// connection. connectionFlag is the value of --connection.
	ForLogin(url, connectionFlag string) (Connection, error)

	// Save stores connection under its name and returns the connection that
	// was stored under that name before.
	Save(connection Connection) (Connection, error)

	// SignOut removes the token of connection, if it is still the same
	// token. The connection stays.
	SignOut(connection Connection) error

	// List returns all saved connections, sorted by name.
	List() ([]Connection, error)

	// Remove deletes the connection that is saved under name and returns it.
	Remove(name string) (Connection, error)
}

// NewStore returns the Store that keeps the connections in the file
// tidegate/connections.json in the user's configuration directory.
func NewStore() Store {
	return fileStore{}
}

type fileStore struct{}

func (fileStore) Current(connectionFlag string) (Connection, error) {
	serverVariable := os.Getenv("TIDEGATE_SERVER")
	tokenVariable := os.Getenv("TIDEGATE_TOKEN")
	usesVariables := serverVariable != "" || tokenVariable != ""

	switch {
	case usesVariables && connectionFlag != "":
		return Connection{}, errors.New("give --connection, or TIDEGATE_SERVER and TIDEGATE_TOKEN, not both")
	case usesVariables && serverVariable == "":
		return Connection{}, errors.New("TIDEGATE_TOKEN is set, but TIDEGATE_SERVER isn't. Set both, or unset TIDEGATE_TOKEN to use a saved connection")
	case usesVariables:
		return Connection{Server: serverVariable, Token: secret.New(tokenVariable)}, nil
	}

	savedByName, err := readFile()
	if err != nil {
		return Connection{}, err
	}

	return findSaved(savedByName, connectionFlag)
}

func (fileStore) ForLogin(url, connectionFlag string) (Connection, error) {
	savedByName, err := readFile()
	if err != nil {
		return Connection{}, err
	}

	if url == "" {
		signInAgain, err := findSaved(savedByName, connectionFlag)
		if err != nil {
			return Connection{}, err
		}

		return Connection{Name: signInAgain.Name, Server: signInAgain.Server}, nil
	}

	name, err := nameForLogin(savedByName, url, connectionFlag)
	if err != nil {
		return Connection{}, err
	}

	return Connection{Name: name, Server: url}, nil
}

func (fileStore) Save(connection Connection) (Connection, error) {
	var replaced Connection

	err := changeFile(func(savedByName map[string]Connection) (bool, error) {
		err := nameSavedForAnotherServer(savedByName, connection.Name, connection.Server)
		if err != nil {
			return false, err
		}

		replaced = savedByName[connection.Name]
		savedByName[connection.Name] = connection

		return true, nil
	})

	return replaced, err
}

func (fileStore) SignOut(connection Connection) error {
	return changeFile(func(savedByName map[string]Connection) (bool, error) {
		saved, isSaved := savedByName[connection.Name]
		if !isSaved || saved.Token.Reveal() != connection.Token.Reveal() {
			return false, nil
		}

		savedByName[connection.Name] = Connection{Name: saved.Name, Server: saved.Server}

		return true, nil
	})
}

func (fileStore) List() ([]Connection, error) {
	savedByName, err := readFile()
	if err != nil {
		return nil, err
	}

	sortedByName := make([]Connection, 0, len(savedByName))
	for _, name := range slices.Sorted(maps.Keys(savedByName)) {
		sortedByName = append(sortedByName, savedByName[name])
	}

	return sortedByName, nil
}

func (fileStore) Remove(name string) (Connection, error) {
	var removed Connection

	err := changeFile(func(savedByName map[string]Connection) (bool, error) {
		saved, isSaved := savedByName[name]
		if !isSaved {
			return false, noConnectionNamed(savedByName, name)
		}

		removed = saved

		delete(savedByName, name)

		return true, nil
	})

	return removed, err
}

// findSaved returns the saved connection that the user means. That is the one
// named by connectionFlag, without it the one named in TIDEGATE_CONNECTION,
// and without that the only saved connection.
func findSaved(savedByName map[string]Connection, connectionFlag string) (Connection, error) {
	wantedName := cmp.Or(connectionFlag, os.Getenv("TIDEGATE_CONNECTION"))
	wanted, isSaved := savedByName[wantedName]

	switch {
	case isSaved:
		return wanted, nil
	case wantedName != "" && len(savedByName) == 0:
		return Connection{}, fmt.Errorf(
			"%w. To save it, run `tidegate login <url> --connection %q`",
			noConnectionNamed(savedByName, wantedName), wantedName,
		)
	case wantedName != "":
		return Connection{}, noConnectionNamed(savedByName, wantedName)
	case len(savedByName) == 0:
		return Connection{}, errors.New("no saved connections. Run `tidegate login <url>` to save one")
	case len(savedByName) == 1:
		onlyName := slices.Collect(maps.Keys(savedByName))[0]

		return savedByName[onlyName], nil
	default:
		return Connection{}, fmt.Errorf(
			"several saved connections: %s. Choose one with --connection or TIDEGATE_CONNECTION",
			quotedNames(savedByName),
		)
	}
}

// nameForLogin returns the name under which a login at url is saved. That is
// connectionFlag if the user gave it. Without it, it is the name of the only
// saved connection that already has this URL, and the URL itself if none has
// it. A name that is saved for another URL is refused.
func nameForLogin(savedByName map[string]Connection, url, connectionFlag string) (string, error) {
	name := connectionFlag

	if name == "" {
		savedWithThisURL := map[string]Connection{}

		for savedName, saved := range savedByName {
			if saved.Server == url {
				savedWithThisURL[savedName] = saved
			}
		}

		switch len(savedWithThisURL) {
		case 0:
			name = url
		case 1:
			return slices.Collect(maps.Keys(savedWithThisURL))[0], nil
		default:
			return "", fmt.Errorf("several saved connections use %s: %s. Choose one with --connection", url, quotedNames(savedWithThisURL))
		}
	}

	err := nameSavedForAnotherServer(savedByName, name, url)
	if err != nil {
		return "", err
	}

	return name, nil
}

// nameSavedForAnotherServer returns an error if name is saved for a server
// other than server.
func nameSavedForAnotherServer(savedByName map[string]Connection, name, server string) error {
	savedUnderName, nameIsTaken := savedByName[name]
	if nameIsTaken && savedUnderName.Server != server {
		return fmt.Errorf(
			"the connection %q is saved for %q. Choose another name, or remove it first with `tidegate connections remove %q`",
			name, savedUnderName.Server, name,
		)
	}

	return nil
}

// noConnectionNamed returns the error for a name that isn't saved. It lists
// the names that are.
func noConnectionNamed(savedByName map[string]Connection, name string) error {
	if len(savedByName) == 0 {
		return fmt.Errorf("no saved connection named %q", name)
	}

	return fmt.Errorf("no saved connection named %q. Saved connections: %s", name, quotedNames(savedByName))
}

// quotedNames returns the names of the connections, sorted, each in quotes,
// with commas between them.
func quotedNames(savedByName map[string]Connection) string {
	names := slices.Sorted(maps.Keys(savedByName))
	for position, name := range names {
		names[position] = strconv.Quote(name)
	}

	return strings.Join(names, ", ")
}

// fileContent is the whole file. It keeps the connections by name.
type fileContent struct {
	Connections map[string]storedConnection `json:"connections"`
}

// storedConnection is how a Connection is written to the file. The name is
// the key in fileContent, so it isn't a field here.
type storedConnection struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

// readFile returns the connections in the file by name. Before the first
// login there is no file, and readFile returns no connections.
func readFile() (map[string]Connection, error) {
	path, err := filePath()
	if err != nil {
		return nil, err
	}

	err = errors.Join(checkOnlyOwnerHasAccess(filepath.Dir(path), true), checkOnlyOwnerHasAccess(path, false))
	if err != nil {
		return nil, err
	}

	fileText, err := os.ReadFile(path) //nolint:gosec // The path is the user's own configuration file.
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]Connection{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading the saved connections: %w", err)
	}

	var content fileContent

	err = json.Unmarshal(fileText, &content)
	if err != nil {
		return nil, fmt.Errorf("reading the saved connections in %s: %w", path, err)
	}

	savedByName := make(map[string]Connection, len(content.Connections))
	for name, stored := range content.Connections {
		savedByName[name] = Connection{Name: name, Server: stored.Server, Token: secret.New(stored.Token)}
	}

	return savedByName, nil
}

const lockWaitLimit = 10 * time.Second

// changeFile reads the connections from the file, lets change edit them, and
// writes them back if change reports that it changed them. It creates the
// directory of the file so that only the owner can use it.
//
// It holds a lock file next to the file meanwhile. Two tidegate processes
// that change the file at once take turns, and one gives up after it waited
// 10 seconds for the other.
func changeFile(change func(savedByName map[string]Connection) (changed bool, err error)) (err error) {
	path, err := filePath()
	if err != nil {
		return err
	}

	directory := filepath.Dir(path)

	err = os.MkdirAll(directory, 0o700)
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
	}

	err = checkOnlyOwnerHasAccess(directory, true)
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

	savedByName, err := readFile()
	if err != nil {
		return err
	}

	changed, err := change(savedByName)
	if err != nil || !changed {
		return err
	}

	return writeFile(path, savedByName)
}

// writeFile replaces the file at path with the given connections. It writes a
// temporary file and renames it, so a reader sees the old file or the new one
// and never a part of it.
func writeFile(path string, savedByName map[string]Connection) error {
	content := fileContent{Connections: make(map[string]storedConnection, len(savedByName))}
	for name, saved := range savedByName {
		content.Connections[name] = storedConnection{Server: saved.Server, Token: saved.Token.Reveal()}
	}

	fileText, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
	}

	// os.CreateTemp creates the file with mode 0600.
	temporaryFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
	}

	_, err = temporaryFile.Write(append(fileText, '\n'))

	err = errors.Join(err, temporaryFile.Close())
	if err == nil {
		err = os.Rename(temporaryFile.Name(), path)
	}

	if err != nil {
		_ = os.Remove(temporaryFile.Name())

		return fmt.Errorf("saving the connections: %w", err)
	}

	return nil
}

// filePath returns where the connections are saved: tidegate/connections.json
// in the user's configuration directory.
func filePath() (string, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding the directory for the saved connections: %w", err)
	}

	if !filepath.IsAbs(configDirectory) {
		return "", fmt.Errorf(
			"the directory for the saved connections, %q, isn't an absolute path. Set HOME or XDG_CONFIG_HOME to an absolute path",
			configDirectory,
		)
	}

	return filepath.Join(configDirectory, "tidegate", "connections.json"), nil
}
