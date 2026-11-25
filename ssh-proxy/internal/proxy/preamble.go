package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

const (
	preambleVersion = "tidegate-session-v1"

	// A ticket has to fit in a cookie, which browsers limit to 4096 bytes.
	maxPreambleBytes = 4096
)

var errInvalidPreamble = errors.New("the connection doesn't start with a preamble that names a ticket")

// readPreamble reads the line that comes before the client's SSH connection,
// which is the version, a space, the ticket and a line break. It returns the
// ticket and a reader that holds what follows the line. A line that is no
// preamble or is longer than maxPreambleBytes fails with an error that wraps
// errInvalidPreamble and never holds the line.
func readPreamble(connection io.Reader) (secret.String, *bufio.Reader, error) {
	reader := bufio.NewReaderSize(connection, maxPreambleBytes)

	line, err := reader.ReadSlice('\n')
	if err != nil {
		return secret.String{}, nil, fmt.Errorf("%w: %w", errInvalidPreamble, err)
	}

	version, ticket, found := strings.Cut(string(line[:len(line)-1]), " ")
	if !found || version != preambleVersion || !isTicket(ticket) {
		return secret.String{}, nil, errInvalidPreamble
	}

	return secret.New(ticket), reader, nil
}

func isTicket(text string) bool {
	if text == "" {
		return false
	}

	for _, character := range []byte(text) {
		if character < '!' || character > '~' {
			return false
		}
	}

	return true
}

// preambleConn reads through the reader that read the preamble, which holds
// the first bytes of the SSH connection.
type preambleConn struct {
	net.Conn

	reader *bufio.Reader
}

func (c *preambleConn) Read(buffer []byte) (int, error) {
	return c.reader.Read(buffer)
}
