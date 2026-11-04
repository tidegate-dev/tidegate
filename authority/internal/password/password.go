// Package password hashes passwords with argon2id.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
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
	// ErrMalformedHash means a hash isn't an argon2id PHC string that Verify
	// can read.
	ErrMalformedHash = errors.New("the password hash isn't a valid argon2id hash")
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

// Verify reports whether password matches hash, a PHC string from Hash. It
// takes the parameters from hash, so hashes made with cheaper parameters
// still work. It returns ErrTooLong for a password over MaxBytes, before hashing,
// and ErrMalformedHash for a hash it can't read. MinLength doesn't apply,
// because a short password just doesn't match.
func Verify(password, hash string) (bool, error) {
	if len(password) > MaxBytes {
		return false, ErrTooLong
	}

	params, err := parse(hash)
	if err != nil {
		return false, err
	}

	key := argon2.IDKey([]byte(normalize(password)), params.salt,
		params.iterations, params.memoryKiB, params.threads, uint32(len(params.key))) //nolint:gosec // parse limits the key to maxKeyLength.

	return subtle.ConstantTimeCompare(key, params.key) == 1, nil
}

type parsedHash struct {
	memoryKiB, iterations uint32
	threads               uint8
	salt, key             []byte
}

// Verify refuses a hash that costs more than one from Hash, so a caller can
// rely on Hash's parameters to bound the cost of a check.
const (
	maxMemoryKiB  = memoryKiB
	maxIterations = iterations
	maxKeyLength  = keyLength
)

func parse(hash string) (parsedHash, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" ||
		parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return parsedHash{}, ErrMalformedHash
	}

	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return parsedHash{}, ErrMalformedHash
	}

	memoryKiB, err := parseParam(params[0], "m")
	if err != nil || memoryKiB > maxMemoryKiB {
		return parsedHash{}, ErrMalformedHash
	}

	iterations, err := parseParam(params[1], "t")
	if err != nil || iterations == 0 || iterations > maxIterations {
		return parsedHash{}, ErrMalformedHash
	}

	threads, err := parseParam(params[2], "p")
	if err != nil || threads == 0 || threads > math.MaxUint8 {
		return parsedHash{}, ErrMalformedHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return parsedHash{}, ErrMalformedHash
	}

	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) == 0 || len(key) > maxKeyLength {
		return parsedHash{}, ErrMalformedHash
	}

	return parsedHash{
		memoryKiB:  memoryKiB,
		iterations: iterations,
		threads:    uint8(threads),
		salt:       salt,
		key:        key,
	}, nil
}

// parseParam reads a parameter like m=65536.
func parseParam(param, name string) (uint32, error) {
	value, found := strings.CutPrefix(param, name+"=")
	if !found {
		return 0, ErrMalformedHash
	}

	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, ErrMalformedHash
	}

	return uint32(parsed), nil
}
