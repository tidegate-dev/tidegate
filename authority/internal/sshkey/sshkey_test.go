package sshkey

import (
	"crypto/ed25519"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestCheckPrivateKeyAcceptsAKeyWithoutAPassphrase(t *testing.T) {
	t.Parallel()

	// arrange
	_, key, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	block, err := ssh.MarshalPrivateKey(key, "")
	require.NoError(t, err)

	// act
	err = CheckPrivateKey(secret.NewBytes(pem.EncodeToMemory(block)))

	// assert
	assert.NoError(t, err)
}

func TestCheckPrivateKeyRefusesWhatIsNoPrivateKey(t *testing.T) {
	t.Parallel()

	for description, privateKey := range map[string]string{
		"nothing":      "",
		"a text":       "not a private key",
		"a public key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5",
		"a PEM block of another type": string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: []byte("not a certificate"),
		})),
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			err := CheckPrivateKey(secret.NewBytes([]byte(privateKey)))

			// assert
			assert.ErrorIs(t, err, ErrPrivateKeyInvalid)
		})
	}
}

func TestCheckPrivateKeyRefusesAnEncryptedPEMBlock(t *testing.T) {
	t.Parallel()

	// arrange
	privateKey := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-128-CBC,00000000000000000000000000000000"},
		Bytes:   []byte("the encrypted key"),
	})

	// act
	err := CheckPrivateKey(secret.NewBytes(privateKey))

	// assert
	assert.ErrorIs(t, err, ErrPrivateKeyEncrypted)
}
