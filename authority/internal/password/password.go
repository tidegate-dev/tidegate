// Package password hashes passwords with argon2id.
package password

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

// The second recommended option of RFC 9106, section 4, for when 2 GiB of
// memory per hash is too much.
const (
	iterations = 3
	memoryKiB  = 64 * 1024
	threads    = 4
	saltLength = 16
	keyLength  = 32
)

// MinLength is the minimum number of characters in a password. NIST
// SP 800-63B sets it for a password that's used alone.
const MinLength = 15

// MaxBytes is the maximum length of a password in bytes. It bounds the work
// of hashing a password.
const MaxBytes = 1024

var (
	// ErrTooShort means a password has fewer than MinLength characters.
	ErrTooShort = fmt.Errorf("the password must have at least %d characters", MinLength)
	// ErrTooLong means a password has more than MaxBytes bytes.
	ErrTooLong = fmt.Errorf("the password must have at most %d bytes", MaxBytes)
)

// Hash returns password's argon2id hash with a new random salt, as a PHC
// string like $argon2id$v=19$m=65536,t=3,p=4$salt$hash. It returns
// ErrTooShort or ErrTooLong for a password it doesn't accept. It hashes the
// password's NFKC normalization, and MinLength counts its characters.
func Hash(password string) (string, error) {
	salt := make([]byte, saltLength)

	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("generating a salt: %w", err)
	}

	return hashWithSalt(password, salt)
}

func hashWithSalt(password string, salt []byte) (string, error) {
	password, err := prepare(password)
	if err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, threads, keyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, iterations, threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// prepare checks password against the rules and returns the form to hash.
func prepare(password string) (string, error) {
	// Before normalizing, because the limit bounds the work.
	if len(password) > MaxBytes {
		return "", ErrTooLong
	}

	password = normalize(password)

	if utf8.RuneCountInString(password) < MinLength {
		return "", ErrTooShort
	}

	return password, nil
}

// Equivalent reports whether a and b are the same password once normalized,
// so they get the same hash.
func Equivalent(a, b string) bool {
	return normalize(a) == normalize(b)
}

// normalize follows NIST SP 800-63B, so a password matches however the
// client encodes it, for example é as one code point or as e with a
// combining accent.
func normalize(password string) string {
	return norm.NFKC.String(password)
}
