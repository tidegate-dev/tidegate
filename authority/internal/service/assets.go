package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/sshkey"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/secretengine"
)

const deleteStoredValueTimeout = 10 * time.Second

// ErrSSHPasswordEmpty means the password for an SSH target has no characters.
var ErrSSHPasswordEmpty = errors.New("the SSH password is empty")

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
// projectName and returns it. It stores the password or private key from
// secretInput in the secret engine and records on the asset where it is.
//
// It returns ErrSSHPasswordEmpty, sshkey.ErrPrivateKeyInvalid or
// sshkey.ErrPrivateKeyEncrypted for a password or private key that tidegate
// can't use, model.ErrProjectNotFound if no project has that name, and
// model.ErrAssetNameTaken if another asset of the project has the name. The
// API checks the form of every other value before it calls CreateSSH.
func (a *Assets) CreateSSH(
	ctx context.Context,
	projectName string,
	name string,
	labels map[string]string,
	target model.SSHTarget,
	secretInput SSHSecretInput,
) (*model.Asset, error) {
	err := checkSSHSecretInput(secretInput)
	if err != nil {
		return nil, err
	}

	project, err := a.projects.FindByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	id, err := model.NewAssetID()
	if err != nil {
		return nil, err
	}

	engine, err := a.engines.ByName(a.engineName)
	if err != nil {
		return nil, err
	}

	reference, err := engine.Store(ctx, secretInput.Value)
	if err != nil {
		return nil, err
	}

	target.Secret = model.StoredSSHSecret{
		EngineName: a.engineName,
		Reference:  string(reference),
		Kind:       secretInput.Kind,
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
		deleteStoredValue(ctx, engine, a.engineName, reference)

		return nil, err
	}

	return asset, nil
}

func checkSSHSecretInput(secretInput SSHSecretInput) error {
	switch secretInput.Kind {
	case model.SSHSecretKindPassword:
		if len(secretInput.Value.Reveal()) == 0 {
			return ErrSSHPasswordEmpty
		}

		return nil
	case model.SSHSecretKindPrivateKey:
		return sshkey.CheckPrivateKey(secretInput.Value)
	default:
		return fmt.Errorf("unknown SSH secret kind %q", secretInput.Kind)
	}
}

func deleteStoredValue(ctx context.Context, engine secretengine.Engine, engineName string, reference secretengine.Reference) {
	// The commit also fails when the request's context ends.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deleteStoredValueTimeout)
	defer cancel()

	err := engine.Delete(ctx, reference)
	if err != nil {
		logging.FromContext(ctx).ErrorContext(ctx, "a secret engine keeps a value that no asset uses, because deleting it failed",
			"engine", engineName, "reference", reference, "error", err)
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
