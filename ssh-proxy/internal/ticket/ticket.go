// Package ticket reads admission tickets. The authority issues one for each
// session, and the ssh-proxy checks it before it calls the authority.
package ticket

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

// ErrInvalid means that a ticket can't be used. It doesn't say why, because
// the reason helps someone who guesses tickets.
var ErrInvalid = errors.New("invalid ticket")

const fingerprintPrefix = "SHA256:"

// Claims are the parts of an admission ticket that the ssh-proxy acts on.
type Claims struct {
	// ClientKeyFingerprint is the SHA256 fingerprint of the SSH public key
	// that the client has to hold, like "SHA256:nThbg6kXUpJW...".
	ClientKeyFingerprint string
}

// Verifier checks an admission ticket without calling the authority.
type Verifier interface {
	// Verify returns the claims of the ticket. If the ticket can't be used,
	// the error wraps ErrInvalid.
	Verify(ticket secret.String) (Claims, error)
}

// Unsigned takes a ticket that consists of the fingerprint of the client's
// key and nothing else. Nobody signed such a ticket, so everyone can write
// one for their own key.
type Unsigned struct{}

// Verify returns the fingerprint that the ticket consists of.
func (Unsigned) Verify(ticket secret.String) (Claims, error) {
	fingerprint := ticket.Reveal()

	encoded, found := strings.CutPrefix(fingerprint, fingerprintPrefix)
	if !found {
		return Claims{}, ErrInvalid
	}

	hash, err := base64.RawStdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(hash) != sha256.Size {
		return Claims{}, ErrInvalid
	}

	return Claims{ClientKeyFingerprint: fingerprint}, nil
}
