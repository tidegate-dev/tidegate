package service

import (
	"context"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// AssetLister lists the assets of a project.
type AssetLister interface {
	// InProject returns the assets of the project, ordered by name.
	InProject(ctx context.Context, projectID model.ProjectID) ([]*model.Asset, error)
}

// Assets creates and lists assets.
type Assets struct {
	projects ProjectFinder
	assets   AssetLister
	units    model.UnitFactory
}

// NewAssets returns Assets that find projects in projects, read assets from
// assets and write in units of work from units.
func NewAssets(projects ProjectFinder, assets AssetLister, units model.UnitFactory) *Assets {
	return &Assets{projects: projects, assets: assets, units: units}
}

// CreateSSH creates an asset for an SSH host in the project named
// projectName and returns it. It returns model.ErrProjectNotFound if no
// project has that name, and model.ErrAssetNameTaken if another asset of the
// project has the name. The API checks the form of every value before it
// calls CreateSSH.
func (a *Assets) CreateSSH(
	ctx context.Context,
	projectName string,
	name string,
	labels map[string]string,
	target model.SSHTarget,
) (*model.Asset, error) {
	project, err := a.projects.ByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	id, err := model.NewAssetID()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return asset, nil
}

// List returns the assets of the project named projectName, ordered by name.
// It returns model.ErrProjectNotFound if no project has that name.
func (a *Assets) List(ctx context.Context, projectName string) ([]*model.Asset, error) {
	project, err := a.projects.ByName(ctx, projectName)
	if err != nil {
		return nil, err
	}

	return a.assets.InProject(ctx, project.ID)
}
