package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/secretengine"
)

const deleteStoredValueTimeout = 10 * time.Second

var (
	// ErrSSHPasswordEmpty means the password for an SSH target has no
	// characters.
	ErrSSHPasswordEmpty = errors.New("the SSH password is empty")
	// ErrSSHPrivateKeyInvalid means the private key for an SSH target isn't a
	// private key in a format that tidegate reads.
	ErrSSHPrivateKeyInvalid = errors.New("the SSH private key isn't a private key that tidegate can read")
	// ErrSSHPrivateKeyEncrypted means the private key for an SSH target needs
	// a passphrase.
	ErrSSHPrivateKeyEncrypted = errors.New("the SSH private key has a passphrase, which tidegate can't use")
)

// AssetLister lists the assets of a project.
type AssetLister interface {
	// ListForProjectID returns the assets of the project, ordered by name.
	ListForProjectID(ctx context.Context, projectID model.ProjectID) ([]*model.Asset, error)
}

// SSHSecretInput is the password or private key that the user of a new SSH
// asset logs in with.
type SSHSecretInput struct {
	Kind model.SSHSecretKind
	// Value is the password or the private key, as Kind says.
	Value secret.Bytes
}

// Assets creates and lists assets.
type Assets struct {
	projects   ProjectFinder
	assets     AssetLister
	units      model.UnitFactory
	engines    *secretengine.Engines
	engineName string
}

// NewAssets returns Assets that find projects in projects, read assets from
// assets and write in units of work from units. They store the password or
// private key for a new SSH asset in the engine that engines has under
// engineName.
func NewAssets(
	projects ProjectFinder,
	assets AssetLister,
	units model.UnitFactory,
	engines *secretengine.Engines,
	engineName string,
) *Assets {
	return &Assets{projects: projects, assets: assets, units: units, engines: engines, engineName: engineName}
}

// CreateSSH creates an asset for an SSH host in the project named
// projectName and returns it. If secretInput isn't nil, it stores the
// password or private key in the secret engine and records on the asset
// where it is. It returns model.ErrProjectNotFound if no project has that
// name, and model.ErrAssetNameTaken if another asset of the project has the
// name. The API checks the form of every value before it calls CreateSSH.
func (a *Assets) CreateSSH(
	ctx context.Context,
	projectName string,
	name string,
	labels map[string]string,
	target model.SSHTarget,
	secretInput *SSHSecretInput,
) (*model.Asset, error) {
	if secretInput != nil {
		err := checkSSHSecretInput(secretInput)
		if err != nil {
			return nil, err
		}
	}

	project, err := a.projects.FindByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	id, err := model.NewAssetID()
	if err != nil {
		return nil, err
	}

	if secretInput != nil {
		target.Secret, err = a.storeValue(ctx, secretInput)
		if err != nil {
			return nil, err
		}
	}

	asset := &model.Asset{
		ID:        id,
		ProjectID: project.ID,
		Name:      name,
		Kind:      model.AssetKindSSH,
		Labels:    labels,
		SSH:       &target,
	}

	unit := a.units.NewUnit()
	unit.Assets().Add(asset)

	err = unit.Commit(ctx)
	if err != nil {
		a.deleteStoredValue(ctx, target.Secret)

		return nil, err
	}

	return asset, nil
}

func checkSSHSecretInput(secretInput *SSHSecretInput) error {
	switch secretInput.Kind {
	case model.SSHSecretKindPassword:
		if len(secretInput.Value.Reveal()) == 0 {
			return ErrSSHPasswordEmpty
		}

		return nil
	case model.SSHSecretKindPrivateKey:
		_, err := ssh.ParseRawPrivateKey(secretInput.Value.Reveal())

		var passphraseMissing *ssh.PassphraseMissingError
		if errors.As(err, &passphraseMissing) {
			return ErrSSHPrivateKeyEncrypted
		}

		if err != nil {
			return ErrSSHPrivateKeyInvalid
		}

		return nil
	default:
		return fmt.Errorf("unknown SSH secret kind %q", secretInput.Kind)
	}
}

func (a *Assets) storeValue(ctx context.Context, secretInput *SSHSecretInput) (*model.StoredSSHSecret, error) {
	engine, err := a.engines.ByName(a.engineName)
	if err != nil {
		return nil, err
	}

	reference, err := engine.Store(ctx, secretInput.Value)
	if err != nil {
		return nil, err
	}

	return &model.StoredSSHSecret{
		EngineName: a.engineName,
		Reference:  string(reference),
		Kind:       secretInput.Kind,
	}, nil
}

func (a *Assets) deleteStoredValue(ctx context.Context, stored *model.StoredSSHSecret) {
	if stored == nil {
		return
	}

	// The commit also fails when the request's context ends.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deleteStoredValueTimeout)
	defer cancel()

	engine, err := a.engines.ByName(stored.EngineName)
	if err == nil {
		err = engine.Delete(ctx, secretengine.Reference(stored.Reference))
	}

	if err != nil {
		slog.ErrorContext(ctx, "a secret engine keeps a value that no asset uses, because deleting it failed",
			"engine", stored.EngineName, "reference", stored.Reference, "error", err)
	}
}

// List returns the assets of the project named projectName, ordered by name.
// It returns model.ErrProjectNotFound if no project has that name.
func (a *Assets) List(ctx context.Context, projectName string) ([]*model.Asset, error) {
	project, err := a.projects.FindByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	return a.assets.ListForProjectID(ctx, project.ID)
}
