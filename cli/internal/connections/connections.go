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
	savedByName, err := readFile()
	if err != nil {
		return Connection{}, err
	}

	replaced := savedByName[connection.Name]
	savedByName[connection.Name] = connection

	err = writeFile(savedByName)
	if err != nil {
		return Connection{}, err
	}

	return replaced, nil
}

func (fileStore) SignOut(connection Connection) error {
	savedByName, err := readFile()
	if err != nil {
		return err
	}

	saved, isSaved := savedByName[connection.Name]
	if !isSaved || saved.Token.Reveal() != connection.Token.Reveal() {
		return nil
	}

	savedByName[connection.Name] = Connection{Name: saved.Name, Server: saved.Server}

	return writeFile(savedByName)
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
	savedByName, err := readFile()
	if err != nil {
		return Connection{}, err
	}

	removed, isSaved := savedByName[name]
	if !isSaved {
		return Connection{}, noConnectionNamed(savedByName, name)
	}

	delete(savedByName, name)

	err = writeFile(savedByName)
	if err != nil {
		return Connection{}, err
	}

	return removed, nil
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

	savedUnderName, nameIsTaken := savedByName[name]
	if nameIsTaken && savedUnderName.Server != url {
		return "", fmt.Errorf(
			"the connection %q is saved for %q. Choose another name, or remove it first with `tidegate connections remove %q`",
			name, savedUnderName.Server, name,
		)
	}

	return name, nil
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

// writeFile replaces the file with the given connections. It creates the file
// and its directory so that only the owner can use them.
func writeFile(savedByName map[string]Connection) error {
	path, err := filePath()
	if err != nil {
		return err
	}

	content := fileContent{Connections: make(map[string]storedConnection, len(savedByName))}
	for name, saved := range savedByName {
		content.Connections[name] = storedConnection{Server: saved.Server, Token: saved.Token.Reveal()}
	}

	fileText, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("saving the connections: %w", err)
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

	err = os.WriteFile(path, append(fileText, '\n'), 0o600)
	if err != nil {
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
