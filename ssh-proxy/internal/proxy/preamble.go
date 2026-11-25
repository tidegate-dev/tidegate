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

// newPreambleReader returns a reader for readPreamble. Its buffer is as long
// as the longest preamble.
func newPreambleReader(connection io.Reader) *bufio.Reader {
	return bufio.NewReaderSize(connection, maxPreambleBytes)
}

// readPreamble reads the line that comes before the client's SSH connection,
// which is the version, a space, the ticket and a line break. It returns the
// ticket. What follows the line stays in reader. A line that is no preamble
// fails with an error that wraps errInvalidPreamble and never holds the line.
func readPreamble(reader *bufio.Reader) (secret.String, error) {
	line, err := reader.ReadSlice('\n')
	if err != nil {
		return secret.String{}, fmt.Errorf("%w: %w", errInvalidPreamble, err)
	}

	version, ticket, found := strings.Cut(string(line[:len(line)-1]), " ")
	if !found || version != preambleVersion || !isTicket(ticket) {
		return secret.String{}, errInvalidPreamble
	}

	return secret.New(ticket), nil
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
