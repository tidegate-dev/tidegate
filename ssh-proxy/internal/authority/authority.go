// Package authority has the calls that the ssh-proxy makes to a phemera
// authority.
package authority

import (
	"context"

	"github.com/phemera-dev/phemera/internal/secret"
)

// SessionID identifies a session at the authority.
type SessionID string

// PreparedSession is what the authority returns for a ticket: the session and
// the SSH host to connect to. It holds nothing to log in with.
type PreparedSession struct {
	SessionID SessionID
	// Host is the SSH host's DNS name or IP address.
	Host string
	Port uint16
	// UserName is the user that the ssh-proxy logs in as on the host.
	UserName string
	// HostKey is the host's public key as the key type and the key in
	// base64, like "ssh-ed25519 AAAAC3...".
	HostKey string
}

// Sessions has the authority's calls for one session, in the order that the
// ssh-proxy makes them. The ssh-proxy logs their errors, so an error must
// hold neither the ticket nor the password.
type Sessions interface {
	// PrepareSession redeems the ticket and returns the session and the SSH
	// host to connect to.
	PrepareSession(ctx context.Context, ticket secret.String) (PreparedSession, error)

	// IssueSessionCredential returns the password to log in to the session's
	// host with. observedHostKey is the public key that the host presented,
	// in the form of PreparedSession.HostKey.
	IssueSessionCredential(ctx context.Context, sessionID SessionID, observedHostKey string) (secret.String, error)
}
