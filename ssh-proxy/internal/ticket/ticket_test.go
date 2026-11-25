package ticket_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/ticket"
)

func TestUnsignedReturnsTheTicketAsTheFingerprintOfTheClientKey(t *testing.T) {
	t.Parallel()

	// arrange
	fingerprint := "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8"

	// act
	claims, err := ticket.Unsigned{}.Verify(secret.New(fingerprint))

	// assert
	require.NoError(t, err)
	assert.Equal(t, ticket.Claims{ClientKeyFingerprint: fingerprint}, claims)
}
