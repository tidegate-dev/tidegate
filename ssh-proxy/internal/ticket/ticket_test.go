package ticket_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/ticket"
)

const fingerprint = "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8"

func TestUnsignedReturnsTheFingerprintThatTheTicketConsistsOf(t *testing.T) {
	t.Parallel()

	// act
	claims, err := ticket.Unsigned{}.Verify(secret.New(fingerprint))

	// assert
	require.NoError(t, err)
	assert.Equal(t, ticket.Claims{ClientKeyFingerprint: fingerprint}, claims)
}

func TestUnsignedRefusesATicketThatIsNoSHA256Fingerprint(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"empty":                "",
		"without the prefix":   "nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8",
		"MD5 fingerprint":      "MD5:16:27:ac:a5:76:28:2d:36:63:1b:56:4d:eb:df:a6:48",
		"too short":            "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5",
		"too long":             "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8AAAA",
		"with padding":         "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8=",
		"not base64":           "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY!",
		"text after the print": "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8 x",
	}

	for name, text := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := ticket.Unsigned{}.Verify(secret.New(text))

			// assert
			require.ErrorIs(t, err, ticket.ErrInvalid)
		})
	}
}
