package api

import (
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestCreateAssetRequestAcceptsAnSSHHostWithLabels(t *testing.T) {
	t.Parallel()

	for description, labels := range map[string]map[string]string{
		"no labels":            nil,
		"plain key":            {"env": "prod"},
		"key with a DNS name":  {"example.com/team": "payments"},
		"empty value":          {"canary": ""},
		"dots and underscores": {"app.kubernetes_io": "v1.2_3"},
		"63 characters":        {strings.Repeat("k", 63): strings.Repeat("v", 63)},
		"DNS name of 253":      {dnsName(253) + "/team": "payments"},
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// arrange
			request := validCreateAssetRequest()
			request.Labels = labels

			// act
			err := validationError(t, request)

			// assert
			assert.NoError(t, err)
		})
	}
}

func TestCreateAssetRequestRefusesWhatBreaksARule(t *testing.T) {
	t.Parallel()

	for description, test := range map[string]struct {
		change  func(request *authorityv1.CreateAssetRequest)
		message string
	}{
		"no target": {
			func(request *authorityv1.CreateAssetRequest) { request.Target = nil },
			"target",
		},
		"empty host": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "" },
			"ssh.host",
		},
		"host with 254 characters": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = strings.Repeat("a", 254) },
			"ssh.host",
		},
		"port 0": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Port = 0 },
			"ssh.port",
		},
		"port 65536": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Port = 65536 },
			"ssh.port",
		},
		"host with a space": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "bastion example" },
			"ssh.host",
		},
		"host with a line break": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "bastion.example\n" },
			"ssh.host",
		},
		"host that starts with a dash": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "-oProxyCommand=x" },
			"ssh.host",
		},
		"host with a port": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "bastion.example:22" },
			"ssh.host",
		},
		"host with a user name": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "root@bastion.example" },
			"ssh.host",
		},
		"host with a slash": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "bastion/example" },
			"ssh.host",
		},
		"host with a wildcard": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "*.example" },
			"ssh.host",
		},
		"IPv6 host with a zone": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "fe80::1%eth0" },
			"ssh.host",
		},
		"IPv6 host in brackets": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().Host = "[2001:db8::7]" },
			"ssh.host",
		},
		"host key with one base64 character": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "ssh-ed25519 A" },
			"ssh.host_key: must be the key type",
		},
		"host key with padding it doesn't need": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "ssh-ed25519 AAAA==" },
			"ssh.host_key: must be the key type",
		},
		"host key with padding in the middle": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "ssh-ed25519 AA==AAAA" },
			"ssh.host_key: must be the key type",
		},
		"host key marked as a certificate authority": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "@cert-authority AAAA" },
			"ssh.host_key: must be the key type",
		},
		"host certificate": {
			func(request *authorityv1.CreateAssetRequest) {
				request.GetSsh().HostKey = "ssh-ed25519-cert-v01@openssh.com AAAA"
			},
			"ssh.host_key: must be the key type",
		},
		"DSA host key": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "ssh-dss AAAA" },
			"ssh.host_key: must be the key type",
		},
		"empty host key": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "" },
			"ssh.host_key: must be the key type",
		},
		"host key without a type": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "AAAA" },
			"ssh.host_key: must be the key type",
		},
		"host key with a comment": {
			func(request *authorityv1.CreateAssetRequest) {
				request.GetSsh().HostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 root@host"
			},
			"ssh.host_key: must be the key type",
		},
		"host key with a second line": {
			func(request *authorityv1.CreateAssetRequest) {
				request.GetSsh().HostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5\nssh-ed25519 AAAAC3NzaC1lZDI1NTE5"
			},
			"ssh.host_key: must be the key type",
		},
		"host key with an escape sequence": {
			func(request *authorityv1.CreateAssetRequest) { request.GetSsh().HostKey = "ssh-ed25519 AAAA\x1b[2J" },
			"ssh.host_key: must be the key type",
		},
		"host key with 8193 characters": {
			func(request *authorityv1.CreateAssetRequest) {
				request.GetSsh().HostKey = "ssh-rsa " + strings.Repeat("A", 8185)
			},
			"ssh.host_key: must be the key type",
		},
		"label key with a space": {
			func(request *authorityv1.CreateAssetRequest) { request.Labels = map[string]string{"my env": "prod"} },
			"labels",
		},
		"label key that starts with a dash": {
			func(request *authorityv1.CreateAssetRequest) { request.Labels = map[string]string{"-env": "prod"} },
			"labels",
		},
		"empty label key": {
			func(request *authorityv1.CreateAssetRequest) { request.Labels = map[string]string{"": "prod"} },
			"labels",
		},
		"label key with 64 characters": {
			func(request *authorityv1.CreateAssetRequest) {
				request.Labels = map[string]string{strings.Repeat("k", 64): "prod"}
			},
			"labels",
		},
		"label key with a DNS label of 64 characters": {
			func(request *authorityv1.CreateAssetRequest) {
				request.Labels = map[string]string{strings.Repeat("a", 64) + ".example/team": "payments"}
			},
			"labels",
		},
		"label key with a DNS name of 254 characters": {
			func(request *authorityv1.CreateAssetRequest) {
				request.Labels = map[string]string{dnsName(254) + "/team": "payments"}
			},
			"labels",
		},
		"label key with two slashes": {
			func(request *authorityv1.CreateAssetRequest) {
				request.Labels = map[string]string{"example.com/team/name": "prod"}
			},
			"labels",
		},
		"label value with a space": {
			func(request *authorityv1.CreateAssetRequest) { request.Labels = map[string]string{"env": "pro d"} },
			"labels",
		},
		"label value with 64 characters": {
			func(request *authorityv1.CreateAssetRequest) {
				request.Labels = map[string]string{"env": strings.Repeat("v", 64)}
			},
			"labels",
		},
		"label value with an escape sequence": {
			func(request *authorityv1.CreateAssetRequest) { request.Labels = map[string]string{"env": "a\x1b[2Jb"} },
			"labels",
		},
		"65 labels": {
			func(request *authorityv1.CreateAssetRequest) {
				request.Labels = map[string]string{}
				for i := range 65 {
					request.Labels[fmt.Sprintf("key%d", i)] = "value"
				}
			},
			"labels",
		},
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// arrange
			request := validCreateAssetRequest()
			test.change(request)

			// act
			err := validationError(t, request)

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, test.message)
		})
	}
}

func TestCreateAssetRequestRefusesAnAssetNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, name := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// arrange
			request := validCreateAssetRequest()
			request.Name = name

			// act
			err := validationError(t, request)

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "name: must start with a lowercase letter")
		})
	}
}

func TestCreateAssetRequestRefusesAProjectNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, project := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// arrange
			request := validCreateAssetRequest()
			request.ProjectName = project

			// act
			err := validationError(t, request)

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "project_name: must start with a lowercase letter")
		})
	}
}

func TestListAssetsRequestRefusesAProjectNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	// act
	err := validationError(t, &authorityv1.ListAssetsRequest{ProjectName: "Payments"})

	// assert
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorContains(t, err, "project_name: must start with a lowercase letter")
}

func TestCreateAssetResponseHoldsTheAssetWithItsSSHTarget(t *testing.T) {
	t.Parallel()

	// arrange
	asset := &model.Asset{
		ID:        model.AssetID{1},
		ProjectID: model.ProjectID{2},
		Name:      "bastion",
		Kind:      model.AssetKindSSH,
		Labels:    map[string]string{"env": "prod"},
		SSH:       &model.SSHTarget{Host: "bastion.example", Port: 2222, HostKey: "ssh-ed25519 AAAA"},
	}

	// act
	response, err := createAssetResponse(asset)

	// assert
	require.NoError(t, err)
	assert.Equal(t, asset.ID.String(), response.GetId())
	assert.Equal(t, asset.ProjectID.String(), response.GetProjectId())
	assert.Equal(t, "bastion", response.GetName())
	assert.Equal(t, map[string]string{"env": "prod"}, response.GetLabels())
	assert.Equal(t, "bastion.example", response.GetSsh().GetHost())
	assert.Equal(t, uint32(2222), response.GetSsh().GetPort())
	assert.Equal(t, "ssh-ed25519 AAAA", response.GetSsh().GetHostKey())
}

func TestCreateAssetResponseFailsForAnAssetWithoutATarget(t *testing.T) {
	t.Parallel()

	// act
	_, err := createAssetResponse(&model.Asset{Name: "bastion"})

	// assert
	assert.ErrorContains(t, err, "has no target")
}

func TestListedAssetHoldsTheAssetWithItsSSHTarget(t *testing.T) {
	t.Parallel()

	// arrange
	asset := &model.Asset{
		ID:        model.AssetID{1},
		ProjectID: model.ProjectID{2},
		Name:      "bastion",
		Kind:      model.AssetKindSSH,
		Labels:    map[string]string{"env": "prod"},
		SSH:       &model.SSHTarget{Host: "bastion.example", Port: 2222, HostKey: "ssh-ed25519 AAAA"},
	}

	// act
	message, err := listedAsset(asset)

	// assert
	require.NoError(t, err)
	assert.Equal(t, asset.ID.String(), message.GetId())
	assert.Equal(t, asset.ProjectID.String(), message.GetProjectId())
	assert.Equal(t, "bastion", message.GetName())
	assert.Equal(t, map[string]string{"env": "prod"}, message.GetLabels())
	assert.Equal(t, "bastion.example", message.GetSsh().GetHost())
	assert.Equal(t, uint32(2222), message.GetSsh().GetPort())
	assert.Equal(t, "ssh-ed25519 AAAA", message.GetSsh().GetHostKey())
}

func TestListedAssetFailsForAnAssetWithoutATarget(t *testing.T) {
	t.Parallel()

	// act
	_, err := listedAsset(&model.Asset{Name: "bastion"})

	// assert
	assert.ErrorContains(t, err, "has no target")
}

func TestCreateAssetRequestAcceptsADNSNameOrAnIPAddressAsTheHost(t *testing.T) {
	t.Parallel()

	for _, host := range []string{
		"bastion.example",
		"localhost",
		"bastion-1.eu.example",
		"192.0.2.7",
		"2001:db8::7",
		dnsName(253),
	} {
		t.Run(host, func(t *testing.T) {
			t.Parallel()

			// arrange
			request := validCreateAssetRequest()
			request.GetSsh().Host = host

			// act
			err := validationError(t, request)

			// assert
			assert.NoError(t, err)
		})
	}
}

func TestCreateAssetRequestAcceptsAHostKeyOfEachPlainKeyType(t *testing.T) {
	t.Parallel()

	for description, hostKey := range map[string]string{
		"Ed25519":                "ssh-ed25519 AAAA",
		"ECDSA with P-256":       "ecdsa-sha2-nistp256 AAAA",
		"ECDSA with P-384":       "ecdsa-sha2-nistp384 AAAA",
		"ECDSA with P-521":       "ecdsa-sha2-nistp521 AAAA",
		"RSA":                    "ssh-rsa AAAA",
		"one padding character":  "ssh-ed25519 AAA=",
		"two padding characters": "ssh-ed25519 AA==",
		"8192 characters":        "ssh-rsa " + strings.Repeat("A", 8184),
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// arrange
			request := validCreateAssetRequest()
			request.GetSsh().HostKey = hostKey

			// act
			err := validationError(t, request)

			// assert
			assert.NoError(t, err)
		})
	}
}

// dnsName returns a DNS name of the length, made of labels of 63 characters.
func dnsName(length int) string {
	name := strings.Repeat(strings.Repeat("a", 63)+".", length/64)

	return name + strings.Repeat("a", length-len(name))
}

func validCreateAssetRequest() *authorityv1.CreateAssetRequest {
	return &authorityv1.CreateAssetRequest{
		ProjectName: "payments",
		Name:        "bastion",
		Target: &authorityv1.CreateAssetRequest_Ssh{Ssh: &authorityv1.SSHTarget{
			Host:    "bastion.example",
			Port:    22,
			HostKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5",
		}},
	}
}

func TestCreateAssetRefusesARequestWhoseTargetTheHandlerDoesNotKnow(t *testing.T) {
	t.Parallel()

	// arrange
	service := NewAssetService(nil)
	request := connect.NewRequest(&authorityv1.CreateAssetRequest{ProjectName: "payments", Name: "bastion"})

	// act
	_, err := service.CreateAsset(t.Context(), request)

	// assert
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
