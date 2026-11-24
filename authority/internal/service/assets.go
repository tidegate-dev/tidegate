package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/secretengine"
)

const deleteLoginValueTimeout = 10 * time.Second

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
