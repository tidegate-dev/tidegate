package password

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestHashMatchesTheReferenceImplementation(t *testing.T) {
	t.Parallel()

	// arrange
	// From the argon2 command of the reference implementation, libargon2:
	// echo -n 'correct horse battery staple' | argon2 tidegate-salt-16 -id -t 3 -m 16 -p 4 -l 32 -e
	const expected = "$argon2id$v=19$m=65536,t=3,p=4$dGlkZWdhdGUtc2FsdC0xNg$mrgC3ZmyavYNNovAISOuUH1wM5aiU+4CgC7QVRPqpCY"

	// act
	actual, err := hashWithSalt("correct horse battery staple", []byte("tidegate-salt-16"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestHashUsesANewSaltEachTime(t *testing.T) {
	t.Parallel()

	// act
	first, err := Hash(secret.New("correct horse battery staple"))
	require.NoError(t, err)

	second, err := Hash(secret.New("correct horse battery staple"))
	require.NoError(t, err)

	// assert
	assert.NotEqual(t, first, second)
	assert.Regexp(t, `^\$argon2id\$v=19\$m=65536,t=3,p=4\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`, first)
}

func TestHashRefusesAPasswordUnder15Characters(t *testing.T) {
	t.Parallel()

	// act
	// 14 characters, but more than 15 bytes in UTF-8.
	_, err := Hash(secret.New("äöüäöüäöüäöüäö"))

	// assert
	require.ErrorIs(t, err, ErrTooShort)
}

func TestHashRefusesAPasswordOver1024Bytes(t *testing.T) {
	t.Parallel()

	// act
	_, err := Hash(secret.New(strings.Repeat("a", 1025)))

	// assert
	require.ErrorIs(t, err, ErrTooLong)
}

func TestHashAcceptsPasswordsAtTheLimits(t *testing.T) {
	t.Parallel()

	for name, password := range map[string]string{
		"15 characters": "äöüäöüäöüäöüäöü",
		"1024 bytes":    strings.Repeat("a", 1024),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := Hash(secret.New(password))

			// assert
			require.NoError(t, err)
		})
	}
}

func TestHashTreatsEquivalentUnicodeFormsAsTheSamePassword(t *testing.T) {
	t.Parallel()

	// arrange
	salt := []byte("tidegate-salt-16")

	// act
	composed, err := hashWithSalt("caf\u00e9 au lait, bitte", salt)
	require.NoError(t, err)

	decomposed, err := hashWithSalt("cafe\u0301 au lait, bitte", salt)
	require.NoError(t, err)

	// assert
	assert.Equal(t, composed, decomposed)
}

func FuzzPrepareTreatsEquivalentUnicodeFormsAsTheSamePassword(f *testing.F) {
	f.Add("caf\u00e9 au lait, bitte")
	f.Add("cafe\u0301 au lait, bitte")
	f.Add("ｃｏｒｒｅｃｔ horse battery")
	f.Add("\ufb01ve boxing wizards jump")
	f.Add("too short")

	f.Fuzz(func(t *testing.T, password string) {
		if len(password) > MaxBytes {
			t.Skip("too long in its original form")
		}

		expected, expectedErr := prepare(password)

		for _, form := range []norm.Form{norm.NFC, norm.NFD, norm.NFKC, norm.NFKD} {
			variant := form.String(password)
			if len(variant) > MaxBytes {
				continue
			}

			actual, err := prepare(variant)

			assert.Equal(t, expected, actual, "form %d of %q", form, password)
			assert.Equal(t, expectedErr, err, "form %d of %q", form, password)
		}
	})
}

func TestEquivalentComparesTheNormalizedForms(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		first, second string
		expected      bool
	}{
		"é in two encodings":  {"café au lait, bitte", "café au lait, bitte", true},
		"different passwords": {"café au lait, bitte", "café au lait, danke", false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			actual := Equivalent(secret.New(test.first), secret.New(test.second))

			// assert
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestVerifyAcceptsTheHashedPassword(t *testing.T) {
	t.Parallel()

	// arrange
	hash, err := Hash(secret.New("correct horse battery staple"))
	require.NoError(t, err)

	// act
	ok, err := Verify(secret.New("correct horse battery staple"), hash)

	// assert
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyRejectsAnotherPassword(t *testing.T) {
	t.Parallel()

	// arrange
	hash, err := Hash(secret.New("correct horse battery staple"))
	require.NoError(t, err)

	// act
	ok, err := Verify(secret.New("correct horse battery stapler"), hash)

	// assert
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerifyReadsTheParametersFromTheHash(t *testing.T) {
	t.Parallel()

	// arrange
	// From the argon2 command of the reference implementation, libargon2:
	// echo -n 'correct horse battery staple' | argon2 tidegate-salt-16 -id -t 1 -m 3 -p 1 -l 16 -e
	const hash = "$argon2id$v=19$m=8,t=1,p=1$dGlkZWdhdGUtc2FsdC0xNg$OtzZSdevu1lF+xEZ6HckGA"

	// act
	ok, err := Verify(secret.New("correct horse battery staple"), hash)

	// assert
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyTreatsEquivalentUnicodeFormsAsTheSamePassword(t *testing.T) {
	t.Parallel()

	// arrange
	hash, err := Hash(secret.New("café au lait, bitte"))
	require.NoError(t, err)

	// act
	ok, err := Verify(secret.New("café au lait, bitte"), hash)

	// assert
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyRejectsAnEmptyPassword(t *testing.T) {
	t.Parallel()

	// arrange
	hash, err := Hash(secret.New("correct horse battery staple"))
	require.NoError(t, err)

	// act
	ok, err := Verify(secret.New(""), hash)

	// assert
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerifyRefusesAPasswordOver1024Bytes(t *testing.T) {
	t.Parallel()

	// arrange
	hash, err := Hash(secret.New("correct horse battery staple"))
	require.NoError(t, err)

	// act
	_, err = Verify(secret.New(strings.Repeat("a", 1025)), hash)

	// assert
	require.ErrorIs(t, err, ErrTooLong)
}

func TestVerifyFailsOnAMalformedHash(t *testing.T) {
	t.Parallel()

	// The library rejects unreadable hashes. These cases cover tidegate's rules.
	for name, hash := range map[string]string{
		"unreadable":                 "",
		"not argon2id":               "$argon2i$v=19$m=65536,t=3,p=4$dGlkZWdhdGUtc2FsdC0xNg$mrgC3ZmyavYNNovAISOuUH1wM5aiU+4CgC7QVRPqpCY",
		"another version":            "$argon2id$v=16$m=65536,t=3,p=4$dGlkZWdhdGUtc2FsdC0xNg$mrgC3ZmyavYNNovAISOuUH1wM5aiU+4CgC7QVRPqpCY",
		"more memory than Hash uses": "$argon2id$v=19$m=131072,t=3,p=4$dGlkZWdhdGUtc2FsdC0xNg$mrgC3ZmyavYNNovAISOuUH1wM5aiU+4CgC7QVRPqpCY",
		"more iterations than Hash":  "$argon2id$v=19$m=65536,t=4,p=4$dGlkZWdhdGUtc2FsdC0xNg$mrgC3ZmyavYNNovAISOuUH1wM5aiU+4CgC7QVRPqpCY",
		"longer key than Hash": "$argon2id$v=19$m=65536,t=3,p=4$dGlkZWdhdGUtc2FsdC0xNg$" +
			base64.RawStdEncoding.EncodeToString(make([]byte, 33)),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := Verify(secret.New("correct horse battery staple"), hash)

			// assert
			require.ErrorIs(t, err, ErrMalformedHash)
		})
	}
}
