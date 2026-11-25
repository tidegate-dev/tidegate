package authority

import (
	"context"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

// DevSSHHost answers every call with the SSH host that `just dev-deps`
// starts. It checks neither the ticket nor the observed host key.
type DevSSHHost struct{}

// PrepareSession returns the SSH host from dev/deps.yaml for every ticket.
func (DevSSHHost) PrepareSession(_ context.Context, _ secret.String) (PreparedSession, error) {
	return PreparedSession{
		SessionID: "00000000-0000-0000-0000-000000000000",
		Host:      "127.0.0.1",
		Port:      2222,
		UserName:  "tidegate",
		HostKey:   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBXe7le0QyMyXW1xMO1Sd7lgeKsvxe/ZFGc8Cp6+7DW0",
	}, nil
}

// IssueSessionCredential returns the password of that host's user for every
// session and host key.
func (DevSSHHost) IssueSessionCredential(_ context.Context, _ SessionID, _ string) (secret.String, error) {
	return secret.New("tidegate"), nil
}
