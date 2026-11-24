package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// The details column stores these keys, so renaming one needs a migration.
type sshDetails struct {
	Host    string `json:"host"`
	Port    uint16 `json:"port"`
	HostKey string `json:"host_key"`
}

// AssetReader queries the assets table on the connection pool.
type AssetReader struct {
	queries *sqlc.Queries
}

// ListForProjectID returns the assets of the project, ordered by name without
// regard to letter case.
func (a *AssetReader) ListForProjectID(ctx context.Context, projectID model.ProjectID) ([]*model.Asset, error) {
	rows, err := a.queries.ListAssetsOfProject(ctx, databaseID(projectID))
	if err != nil {
		return nil, fmt.Errorf("reading the assets: %w", err)
	}

	assets := make([]*model.Asset, len(rows))

	for i, row := range rows {
		assets[i], err = assetFromRow(row)
		if err != nil {
			return nil, err
		}
	}

	return assets, nil
}

func assetFromRow(row sqlc.Asset) (*model.Asset, error) {
	asset := &model.Asset{
		ID:        model.AssetID(row.ID.Bytes),
		ProjectID: model.ProjectID(row.ProjectID.Bytes),
		Name:      row.Name,
		Kind:      model.AssetKind(row.Kind),
	}

	err := json.Unmarshal(row.Labels, &asset.Labels)
	if err != nil {
		return nil, fmt.Errorf("reading the labels of asset %s: %w", asset.ID, err)
	}

	switch asset.Kind {
	case model.AssetKindSSH:
		var details sshDetails

		err = json.Unmarshal(row.Details, &details)
		if err != nil {
			return nil, fmt.Errorf("reading the details of asset %s: %w", asset.ID, err)
		}

		target := model.SSHTarget(details)

		err = checkSSHTarget(&target)
		if err != nil {
			return nil, fmt.Errorf("reading the details of asset %s: %w", asset.ID, err)
		}

		asset.SSH = &target
	default:
		return nil, fmt.Errorf("reading asset %s: unknown kind %q", asset.ID, asset.Kind)
	}

	return asset, nil
}

// AssetReadWriter belongs to a unit of work. It records writes, which the
// unit runs at Commit.
type AssetReadWriter struct {
	unit *Unit
}

// Add records the insert of asset. Commit returns model.ErrAssetNameTaken if
// another asset of the project has the name, ignoring letter case, and
// model.ErrProjectNotFound if the project doesn't exist. It fails for an
// asset whose kind is unknown or whose target is incomplete.
func (a *AssetReadWriter) Add(asset *model.Asset) {
	a.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		details, err := assetDetails(asset)
		if err != nil {
			return fmt.Errorf("creating the asset: %w", err)
		}

		labels := asset.Labels
		if labels == nil {
			labels = map[string]string{}
		}

		encodedLabels, err := json.Marshal(labels)
		if err != nil {
			return fmt.Errorf("creating the asset: writing the labels: %w", err)
		}

		err = queries.InsertAsset(ctx, sqlc.InsertAssetParams{
			ID:        databaseID(asset.ID),
			ProjectID: databaseID(asset.ProjectID),
			Name:      asset.Name,
			Kind:      string(asset.Kind),
			Labels:    encodedLabels,
			Details:   details,
		})

		switch {
		case isUniqueViolationOf(err, "assets_name_unique_in_project"):
			return model.ErrAssetNameTaken
		case isForeignKeyViolationOf(err, "assets_project_fk"):
			return model.ErrProjectNotFound
		case err != nil:
			return fmt.Errorf("creating the asset: %w", err)
		default:
			return nil
		}
	})
}

func assetDetails(asset *model.Asset) ([]byte, error) {
	switch asset.Kind {
	case model.AssetKindSSH:
		err := checkSSHTarget(asset.SSH)
		if err != nil {
			return nil, err
		}

		return json.Marshal(sshDetails(*asset.SSH))
	default:
		return nil, fmt.Errorf("unknown kind %q", asset.Kind)
	}
}

// The details column is JSON, so the table can't require these values.
func checkSSHTarget(target *model.SSHTarget) error {
	if target == nil || target.Host == "" || target.Port == 0 || target.HostKey == "" {
		return errors.New("an SSH asset needs a host, a port and a host key")
	}

	return nil
}
