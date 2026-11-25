package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	// ErrAssetNotFound means no asset has the given ID.
	ErrAssetNotFound = errors.New("asset not found")
	// ErrAssetNameTaken means another asset of the project already has the name.
	// Nothing was written.
	ErrAssetNameTaken = errors.New("the asset name is taken")
)

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

// AssetKindSSH is a host that phemera reaches over SSH.
const AssetKindSSH AssetKind = "ssh"

// Asset is a target in one project that phemera gives access to.
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
// identity with, and the user that phemera logs in as.
type SSHTarget struct {
	Host string
	Port uint16
	// HostKey is the host's public key as the key type and the key in
	// base64, like "ssh-ed25519 AAAAC3...".
	HostKey string
	// UserName is the user that phemera logs in as on the host.
	UserName string
	// Secret says where the user's password or private key is stored.
	Secret StoredSSHSecret
}

// SSHSecretKind says what a user logs in to an SSH host with.
type SSHSecretKind string

const (
	// SSHSecretKindPassword is the user's password.
	SSHSecretKindPassword SSHSecretKind = "password"
	// SSHSecretKindPrivateKey is a private key of the user.
	SSHSecretKindPrivateKey SSHSecretKind = "private_key"
)

// StoredSSHSecret says where a secret engine keeps the password or private
// key of an SSH target's user. It doesn't hold the password or the key.
type StoredSSHSecret struct {
	// EngineName names the secret engine.
	EngineName string
	// Reference names the password or private key within that engine.
	Reference string
	Kind      SSHSecretKind
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
