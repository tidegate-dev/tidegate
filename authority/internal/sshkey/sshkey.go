// Package sshkey checks the host keys of SSH hosts and the private keys that
// phemera logs in to them with.
package sshkey

import (
	"errors"

	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/secret"
)

var (
	// ErrPrivateKeyInvalid means a value isn't a private key in a format that
	// phemera reads.
	ErrPrivateKeyInvalid = errors.New("the SSH private key isn't a private key that phemera can read")
	// ErrPrivateKeyEncrypted means a private key needs a passphrase.
	ErrPrivateKeyEncrypted = errors.New("the SSH private key has a passphrase, which phemera can't use")
	// ErrHostKeyInvalid means a value isn't a public key that phemera can
	// read.
	ErrHostKeyInvalid = errors.New("the SSH host key isn't a public key that phemera can read")
)

// CheckHostKey returns ErrHostKeyInvalid for a host key that isn't a public
// key of the key type in front of it. hostKey is the key type and the key in
// base64, like "ssh-ed25519 AAAAC3...".
func CheckHostKey(hostKey string) error {
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(hostKey))
	if err != nil {
		return ErrHostKeyInvalid
	}

	return nil
}

// CheckPrivateKey returns ErrPrivateKeyEncrypted for a private key that needs
// a passphrase, and ErrPrivateKeyInvalid for any other value that SSH can't
// log in with.
func CheckPrivateKey(privateKey secret.Bytes) error {
	_, err := ssh.ParsePrivateKey(privateKey.Reveal())

	var passphraseMissing *ssh.PassphraseMissingError
	if errors.As(err, &passphraseMissing) {
		return ErrPrivateKeyEncrypted
	}

	if err != nil {
		return ErrPrivateKeyInvalid
	}

	return nil
}
