package command

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestParseAddressReturnsTheHostAndThePort(t *testing.T) {
	t.Parallel()

	for address, want := range map[string]struct {
		host string
		port uint32
	}{
		"bastion.example":      {"bastion.example", 22},
		"bastion.example:2222": {"bastion.example", 2222},
		"192.0.2.7":            {"192.0.2.7", 22},
		"192.0.2.7:65535":      {"192.0.2.7", 65535},
		"2001:db8::7":          {"2001:db8::7", 22},
		"[2001:db8::7]":        {"2001:db8::7", 22},
		"[2001:db8::7]:2222":   {"2001:db8::7", 2222},
	} {
		t.Run(address, func(t *testing.T) {
			t.Parallel()

			// act
			host, port, err := parseAddress(address)

			// assert
			require.NoError(t, err)
			assert.Equal(t, want.host, host)
			assert.Equal(t, want.port, port)
		})
	}
}

func TestParseAddressRefusesWhatIsNoHostWithAnOptionalPort(t *testing.T) {
	t.Parallel()

	for description, address := range map[string]string{
		"empty":            "",
		"only a port":      ":22",
		"empty port":       "bastion.example:",
		"port 0":           "bastion.example:0",
		"port 65536":       "bastion.example:65536",
		"port name":        "bastion.example:ssh",
		"negative port":    "bastion.example:-1",
		"two ports":        "bastion.example:22:22",
		"empty brackets":   "[]:22",
		"unclosed bracket": "[2001:db8::7:22",
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			_, _, err := parseAddress(address)

			// assert
			assert.ErrorContains(t, err, "want an address like host or host:port")
		})
	}
}

func TestParseLabelsReturnsEachKeyWithItsValue(t *testing.T) {
	t.Parallel()

	// act
	labels, err := parseLabels([]string{"env=prod", "canary=", "example.com/team=payments"})

	// assert
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"env": "prod", "canary": "", "example.com/team": "payments"}, labels)
}

func TestParseLabelsRefusesWhatIsNoKeyWithAValue(t *testing.T) {
	t.Parallel()

	for description, test := range map[string]struct {
		pairs   []string
		message string
	}{
		"no equals sign": {[]string{"env"}, `label "env" must look like key=value`},
		"empty key":      {[]string{"=prod"}, `label "=prod" must look like key=value`},
		"key twice":      {[]string{"env=prod", "env=dev"}, `label "env" is given twice`},
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := parseLabels(test.pairs)

			// assert
			assert.EqualError(t, err, test.message)
		})
	}
}

func TestHostKeyWithoutCommentKeepsTheTypeAndTheKey(t *testing.T) {
	t.Parallel()

	for description, text := range map[string]string{
		"type and key":               "ssh-ed25519 AAAAC3Nz",
		"with a comment":             "ssh-ed25519 AAAAC3Nz root@bastion",
		"with a line ending":         "ssh-ed25519 AAAAC3Nz root@bastion\n",
		"with a Windows line ending": "ssh-ed25519 AAAAC3Nz root@bastion\r\n",
		"after an empty line":        "\n  \nssh-ed25519 AAAAC3Nz\n",
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			hostKey, err := hostKeyWithoutComment(text)

			// assert
			require.NoError(t, err)
			assert.Equal(t, "ssh-ed25519 AAAAC3Nz", hostKey)
		})
	}
}

func TestHostKeyWithoutCommentRefusesATextWithoutATypeAndAKey(t *testing.T) {
	t.Parallel()

	for description, text := range map[string]string{
		"empty":    "",
		"one word": "AAAAC3Nz",
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := hostKeyWithoutComment(text)

			// assert
			assert.ErrorContains(t, err, "want a host key like")
		})
	}
}

func TestHostKeyWithoutCommentRefusesSeveralKeyLines(t *testing.T) {
	t.Parallel()

	for description, text := range map[string]string{
		"two lines":                        "ssh-ed25519 AAAAC3Nz\nssh-rsa AAAAB3Nz\n",
		"two lines with a carriage return": "ssh-ed25519 AAAAC3Nz\rssh-rsa AAAAB3Nz",
		"a key and a word":                 "ssh-ed25519 AAAAC3Nz\nmore",
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := hostKeyWithoutComment(text)

			// assert
			assert.ErrorContains(t, err, "want one host key, not several lines")
		})
	}
}

func TestAssetsResultTextShowsTheAddressAndTheLabelsInTheOrderOfTheirKeys(t *testing.T) {
	t.Parallel()

	// arrange
	result := newAssetsResult([]*authorityv1.ListAssetsResponse_Asset{{
		Id:     "1",
		Name:   "bastion",
		Labels: map[string]string{"team": "payments", "env": "prod"},
		Target: &authorityv1.ListAssetsResponse_Asset_Ssh{Ssh: &authorityv1.SSHTarget{Host: "2001:db8::7", Port: 2222}},
	}})

	// act
	text := result.String()

	// assert
	assert.Equal(t, "ID  NAME     ADDRESS             LABELS\n"+
		"1   bastion  [2001:db8::7]:2222  env=prod,team=payments", text)
}

func TestAssetsResultTextNamesTheCreateCommandWhenThereAreNoAssets(t *testing.T) {
	t.Parallel()

	// act
	text := newAssetsResult(nil).String()

	// assert
	assert.Contains(t, text, "tidegate asset create ssh")
}

func TestAssetsResultTextQuotesWhatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// arrange
	result := newAssetsResult([]*authorityv1.ListAssetsResponse_Asset{{
		Id:     "1",
		Name:   "a\x1b[2Jb",
		Labels: map[string]string{"env": "c\x1b[2Jd"},
		Target: &authorityv1.ListAssetsResponse_Asset_Ssh{Ssh: &authorityv1.SSHTarget{Host: "e\x1b[2Jf", Port: 22}},
	}})

	// act
	text := result.String()

	// assert
	assert.NotContains(t, text, "\x1b")
	assert.Contains(t, text, `"a\x1b[2Jb"`)
}

func TestNewAssetResultHasAnEmptySetOfLabelsForAnAssetWithoutLabels(t *testing.T) {
	t.Parallel()

	// act
	result := newAssetResult(&authorityv1.CreateAssetResponse{Name: "bastion"})

	// assert
	assert.NotNil(t, result.Labels)
	assert.Empty(t, result.Labels)
	assert.Nil(t, result.SSH)
}

func TestAssetResultLabelsTextOrdersTheLabelsByKey(t *testing.T) {
	t.Parallel()

	// arrange
	result := assetResult{Labels: map[string]string{"env2": "b", "env": "a"}}

	// act
	text := result.labelsText()

	// assert
	assert.Equal(t, "env=a,env2=b", text)
}

func TestNewSSHTargetResultHoldsTheUserName(t *testing.T) {
	t.Parallel()

	// act
	result := newSSHTargetResult(&authorityv1.SSHTarget{Host: "bastion.example", Port: 22, UserName: "deploy"})

	// assert
	assert.Equal(t, "deploy", result.UserName)
}

func TestSetSSHSecretRefusesFlagsThatDoNotFitTogether(t *testing.T) {
	t.Parallel()

	for description, test := range map[string]struct {
		askForPassword bool
		privateKeyFile string
		message        string
	}{
		"password and private key file": {true, "key", "--password and --private-key-file can't be combined"},
		"neither of the two":            {false, "", "want --password or --private-key-file"},
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// arrange
			asset := &authorityv1.CreateAssetRequest_SSHAsset{}

			// act
			err := setSSHSecret(t.Context(), asset, test.askForPassword, test.privateKeyFile,
				strings.NewReader("a password\n"), io.Discard)

			// assert
			require.ErrorContains(t, err, test.message)
			assert.Nil(t, asset.GetSecret())
		})
	}
}

func TestSetSSHSecretPutsTheContentOfThePrivateKeyFileIntoTheRequest(t *testing.T) {
	t.Parallel()

	// arrange
	privateKeyFile := filepath.Join(t.TempDir(), "key")
	require.NoError(t, os.WriteFile(privateKeyFile, []byte("a private key\n"), 0o600))

	asset := &authorityv1.CreateAssetRequest_SSHAsset{}

	// act
	err := setSSHSecret(t.Context(), asset, false, privateKeyFile, strings.NewReader("a password\n"), io.Discard)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "a private key\n", asset.GetPrivateKey())
}

func TestSetSSHSecretFailsForAPrivateKeyFileThatDoesNotExist(t *testing.T) {
	t.Parallel()

	// arrange
	privateKeyFile := filepath.Join(t.TempDir(), "missing")

	// act
	err := setSSHSecret(t.Context(), &authorityv1.CreateAssetRequest_SSHAsset{}, false, privateKeyFile, strings.NewReader(""), io.Discard)

	// assert
	assert.ErrorContains(t, err, "--private-key-file")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestSetSSHSecretReadsThePasswordFromTheInput(t *testing.T) {
	t.Parallel()

	// arrange
	asset := &authorityv1.CreateAssetRequest_SSHAsset{}

	// act
	err := setSSHSecret(t.Context(), asset, true, "", strings.NewReader("a password\n"), io.Discard)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "a password", asset.GetPassword())
}

func TestSetSSHSecretFailsWithoutAPasswordOnTheInput(t *testing.T) {
	t.Parallel()

	// act
	err := setSSHSecret(t.Context(), &authorityv1.CreateAssetRequest_SSHAsset{}, true, "", strings.NewReader(""), io.Discard)

	// assert
	assert.ErrorContains(t, err, "no password on stdin")
}
