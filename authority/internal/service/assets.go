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

const deleteLoginValueTimeout = 10 * time.Second

var (
	// ErrSSHPasswordEmpty means the password of a login has no characters.
	ErrSSHPasswordEmpty = errors.New("the SSH password is empty")
	// ErrSSHPrivateKeyInvalid means the private key of a login isn't a
	// private key in a format that tidegate reads.
	ErrSSHPrivateKeyInvalid = errors.New("the SSH private key isn't a private key that tidegate can read")
	// ErrSSHPrivateKeyEncrypted means the private key of a login needs a
	// passphrase.
	ErrSSHPrivateKeyEncrypted = errors.New("the SSH private key has a passphrase, which tidegate can't use")
)

// AssetLister lists the assets of a project.
type AssetLister interface {
	// ListForProjectID returns the assets of the project, ordered by name.
	ListForProjectID(ctx context.Context, projectID model.ProjectID) ([]*model.Asset, error)
}

// SSHLoginInput is the login for a new SSH asset.
type SSHLoginInput struct {
	UserName string
	Kind     model.SSHLoginKind
	// Value is the user's password or private key, as Kind says.
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
// private key of a new asset's login in the engine that engines has under
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
// projectName and returns it. If login isn't nil, it stores the login's value
// in the secret engine and the rest of the login with the asset. It returns
// model.ErrProjectNotFound if no project has that name, and
// model.ErrAssetNameTaken if another asset of the project has the name. The
// API checks the form of every value before it calls CreateSSH.
func (a *Assets) CreateSSH(
	ctx context.Context,
	projectName string,
	name string,
	labels map[string]string,
	target model.SSHTarget,
	login *SSHLoginInput,
) (*model.Asset, error) {
	if login != nil {
		err := checkSSHLoginInput(login)
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

	if login != nil {
		target.Login, err = a.storeLogin(ctx, login)
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
		a.deleteLoginValue(ctx, target.Login)

		return nil, err
	}

	return asset, nil
}

func checkSSHLoginInput(login *SSHLoginInput) error {
	switch login.Kind {
	case model.SSHLoginKindPassword:
		if len(login.Value.Reveal()) == 0 {
			return ErrSSHPasswordEmpty
		}

		return nil
	case model.SSHLoginKindPrivateKey:
		_, err := ssh.ParseRawPrivateKey(login.Value.Reveal())

		var passphraseMissing *ssh.PassphraseMissingError
		if errors.As(err, &passphraseMissing) {
			return ErrSSHPrivateKeyEncrypted
		}

		if err != nil {
			return ErrSSHPrivateKeyInvalid
		}

		return nil
	default:
		return fmt.Errorf("unknown login kind %q", login.Kind)
	}
}

func (a *Assets) storeLogin(ctx context.Context, login *SSHLoginInput) (*model.SSHLogin, error) {
	engine, err := a.engines.ByName(a.engineName)
	if err != nil {
		return nil, err
	}

	reference, err := engine.Store(ctx, login.Value)
	if err != nil {
		return nil, err
	}

	return &model.SSHLogin{
		UserName:   login.UserName,
		EngineName: a.engineName,
		Reference:  string(reference),
		Kind:       login.Kind,
	}, nil
}

func (a *Assets) deleteLoginValue(ctx context.Context, login *model.SSHLogin) {
	if login == nil {
		return
	}

	// The commit also fails when the request's context ends.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deleteLoginValueTimeout)
	defer cancel()

	engine, err := a.engines.ByName(login.EngineName)
	if err == nil {
		err = engine.Delete(ctx, secretengine.Reference(login.Reference))
	}

	if err != nil {
		slog.ErrorContext(ctx, "a secret engine keeps a value that no asset uses, because deleting it failed",
			"engine", login.EngineName, "reference", login.Reference, "error", err)
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
