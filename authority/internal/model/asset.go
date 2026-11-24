package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrAssetNameTaken means another asset of the project already has the name.
// Nothing was written.
var ErrAssetNameTaken = errors.New("the asset name is taken")

// AssetID identifies an asset.
type AssetID uuid.UUID

// NewAssetID returns the ID for a new asset.
func NewAssetID() (AssetID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return AssetID{}, fmt.Errorf("creating an asset ID: %w", err)
	}

	return AssetID(id), nil
}

// String returns the ID in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (id AssetID) String() string {
	return uuid.UUID(id).String()
}

// AssetKind names the kind of target that an asset is.
type AssetKind string

// AssetKindSSH is a host that tidegate reaches over SSH.
const AssetKindSSH AssetKind = "ssh"

// Asset is a target in one project that tidegate gives access to.
type Asset struct {
	ID        AssetID
	ProjectID ProjectID
	Name      string
	Kind      AssetKind
	// Labels are the keys and values that a role selects the asset by.
	Labels map[string]string
	// SSH is set if Kind is AssetKindSSH, and nil for every other kind.
	SSH *SSHTarget
}

// SSHTarget is the address of an SSH host, the key that the host proves its
// identity with, and the login that tidegate uses on the host.
type SSHTarget struct {
	Host string
	Port uint16
	// HostKey is the host's public key as the key type and the key in
	// base64, like "ssh-ed25519 AAAAC3...".
	HostKey string
	// Login is nil for a target that has no login.
	Login *SSHLogin
}

// SSHLoginKind names what an SSH host checks at a login.
type SSHLoginKind string

const (
	// SSHLoginKindPassword is a login with the user's password.
	SSHLoginKindPassword SSHLoginKind = "password"
	// SSHLoginKindPrivateKey is a login with a private key of the user.
	SSHLoginKindPrivateKey SSHLoginKind = "private_key"
)

// SSHLogin is the user that tidegate logs in to an SSH host as, and where the
// user's password or private key is stored.
type SSHLogin struct {
	UserName string
	// EngineName names the secret engine that holds the password or private
	// key.
	EngineName string
	// Reference names the password or private key within that engine.
	Reference string
	Kind      SSHLoginKind
}

// AssetReadWriter belongs to a Unit. It records writes, which the unit runs
// at Commit.
type AssetReadWriter interface {
	// Add records the insert of asset. Commit returns ErrAssetNameTaken if
	// another asset of the project has the name, ignoring letter case, and
	// ErrProjectNotFound if the project doesn't exist. It fails for an asset
	// whose kind is unknown or whose target is incomplete.
	Add(asset *Asset)
}
