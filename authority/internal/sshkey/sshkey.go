// Package sshkey checks the private keys that tidegate logs in to SSH hosts
// with.
package sshkey

import (
	"errors"

	"golang.org/x/crypto/ssh"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

var (
	// ErrPrivateKeyInvalid means a value isn't a private key in a format that
	// tidegate reads.
	ErrPrivateKeyInvalid = errors.New("the SSH private key isn't a private key that tidegate can read")
	// ErrPrivateKeyEncrypted means a private key needs a passphrase.
	ErrPrivateKeyEncrypted = errors.New("the SSH private key has a passphrase, which tidegate can't use")
)

// CheckPrivateKey returns ErrPrivateKeyEncrypted for a private key that needs
// a passphrase, and ErrPrivateKeyInvalid for a value that is no private key
// in PEM.
func CheckPrivateKey(privateKey secret.Bytes) error {
	_, err := ssh.ParseRawPrivateKey(privateKey.Reveal())

	var passphraseMissing *ssh.PassphraseMissingError
	if errors.As(err, &passphraseMissing) {
		return ErrPrivateKeyEncrypted
	}

	if err != nil {
		return ErrPrivateKeyInvalid
	}

	return nil
}
