// Package password hashes and checks passwords with argon2id.
package password

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/matthewhartstonge/argon2"
	"golang.org/x/text/unicode/norm"

	"github.com/tidegate-dev/tidegate/internal/secret"
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
	// ErrMalformedHash means Verify can't read or doesn't accept a hash.
	ErrMalformedHash = errors.New("the password hash isn't a valid argon2id hash")
)

// config is the second recommended option of RFC 9106, section 4.
func config() argon2.Config {
	return argon2.MemoryConstrainedDefaults()
}

// Hash returns password's argon2id hash with a new random salt, as a PHC
// string like $argon2id$v=19$m=65536,t=3,p=4$salt$hash. It returns
// ErrTooShort or ErrTooLong for a password it doesn't accept. It hashes the
// password's NFKC normalization, and MinLength counts its characters.
func Hash(password secret.String) (string, error) {
	return hashWithSalt(password.Reveal(), nil)
}

func hashWithSalt(password string, salt []byte) (string, error) {
	password, err := prepare(password)
	if err != nil {
		return "", err
	}

	cfg := config()

	raw, err := cfg.Hash([]byte(password), salt)
	if err != nil {
		return "", fmt.Errorf("hashing the password: %w", err)
	}

	return string(raw.Encode()), nil
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
func Equivalent(a, b secret.String) bool {
	return normalize(a.Reveal()) == normalize(b.Reveal())
}

// normalize follows NIST SP 800-63B, so a password matches however the
// client encodes it, for example é as one code point or as e with a
// combining accent.
func normalize(password string) string {
	return norm.NFKC.String(password)
}

// Verify reports whether password matches hash. It accepts hashes that cost
// less than Hash's, and returns ErrTooLong or ErrMalformedHash.
func Verify(password secret.String, hash string) (bool, error) {
	if len(password.Reveal()) > MaxBytes {
		return false, ErrTooLong
	}

	raw, err := argon2.Decode([]byte(hash))
	if err != nil || !acceptable(raw.Config) {
		return false, ErrMalformedHash
	}

	// The library refuses a nil password.
	ok, err := raw.Verify(append([]byte{}, normalize(password.Reveal())...))
	if err != nil {
		return false, fmt.Errorf("checking the password: %w", err)
	}

	return ok, nil
}

// acceptable checks the version, which the library's Verify ignores.
func acceptable(decoded argon2.Config) bool {
	ours := config()

	return decoded.Mode == argon2.ModeArgon2id &&
		decoded.Version == ours.Version &&
		decoded.MemoryCost <= ours.MemoryCost &&
		decoded.TimeCost <= ours.TimeCost &&
		decoded.HashLength <= ours.HashLength
}
