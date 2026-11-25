// Package ticket reads admission tickets. The authority issues one for each
// session, and the ssh-proxy checks it before it calls the authority.
package ticket

import "github.com/phemera-dev/phemera/internal/secret"

// Claims are the parts of an admission ticket that the ssh-proxy acts on.
type Claims struct {
	// ClientKeyFingerprint is the SHA256 fingerprint of the SSH public key
	// that the client has to hold, like "SHA256:nThbg6kXUpJW...".
	ClientKeyFingerprint string
}

// Verifier checks an admission ticket without calling the authority.
type Verifier interface {
	// Verify returns the claims of the ticket, or an error for a ticket that
	// can't be used. The ssh-proxy logs the error, so it must not hold the
	// ticket.
	Verify(ticket secret.String) (Claims, error)
}

// Unsigned takes a ticket that consists of the fingerprint of the client's
// key and nothing else. Nobody signed such a ticket, so everyone can write
// one for their own key.
type Unsigned struct{}

// Verify returns the ticket's text as the fingerprint of the client's key.
func (Unsigned) Verify(ticket secret.String) (Claims, error) {
	return Claims{ClientKeyFingerprint: ticket.Reveal()}, nil
}
