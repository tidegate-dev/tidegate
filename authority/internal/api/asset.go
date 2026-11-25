package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/service"
	authorityv1 "github.com/phemera-dev/phemera/gen/phemera/authority/v1"
	"github.com/phemera-dev/phemera/internal/secret"
)

// AssetService creates and lists assets.
type AssetService struct {
	assets *service.Assets
}

// NewAssetService returns an AssetService.
func NewAssetService(assets *service.Assets) *AssetService {
	return &AssetService{assets: assets}
}

// CreateAsset creates an asset in a project and returns it.
func (s *AssetService) CreateAsset(
	ctx context.Context,
	req *connect.Request[authorityv1.CreateAssetRequest],
) (*connect.Response[authorityv1.CreateAssetResponse], error) {
	var (
		asset *model.Asset
		err   error
	)

	switch target := req.Msg.GetTarget().(type) {
	case *authorityv1.CreateAssetRequest_Ssh:
		sshAsset := target.Ssh

		sshLogin := sshAsset.GetLogin()

		secretInput, inputErr := newSSHSecretInput(sshLogin)
		if inputErr != nil {
			return nil, inputErr
		}

		sshTarget := sshAsset.GetTarget()

		asset, err = s.assets.CreateSSH(ctx, req.Msg.GetProjectName(), req.Msg.GetName(), req.Msg.GetLabels(), model.SSHTarget{
			Host:     sshTarget.GetHost(),
			Port:     uint16(sshTarget.GetPort()), //nolint:gosec // The API definition allows 1 to 65535.
			HostKey:  sshTarget.GetHostKey(),
			UserName: sshLogin.GetUserName(),
		}, secretInput)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the asset needs a target of a known kind"))
	}

	if err != nil {
		return nil, clientError(err)
	}

	response, err := newCreateAssetResponse(asset)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(response), nil
}

// ListAssets returns every asset of a project.
func (s *AssetService) ListAssets(
	ctx context.Context,
	req *connect.Request[authorityv1.ListAssetsRequest],
) (*connect.Response[authorityv1.ListAssetsResponse], error) {
	assets, err := s.assets.List(ctx, req.Msg.GetProjectName())
	if err != nil {
		return nil, clientError(err)
	}

	messages := make([]*authorityv1.ListAssetsResponse_Asset, len(assets))

	for i, asset := range assets {
		messages[i], err = newListedAsset(asset)
		if err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(&authorityv1.ListAssetsResponse{Assets: messages}), nil
}

func newCreateAssetResponse(asset *model.Asset) (*authorityv1.CreateAssetResponse, error) {
	sshTarget, err := newSSHTarget(asset)
	if err != nil {
		return nil, err
	}

	return &authorityv1.CreateAssetResponse{
		Id:        asset.ID.String(),
		ProjectId: asset.ProjectID.String(),
		Name:      asset.Name,
		Labels:    asset.Labels,
		Target: &authorityv1.CreateAssetResponse_Ssh{Ssh: &authorityv1.CreateAssetResponse_SSHAsset{
			Target:   sshTarget,
			UserName: asset.SSH.UserName,
		}},
	}, nil
}

func newListedAsset(asset *model.Asset) (*authorityv1.ListAssetsResponse_Asset, error) {
	sshTarget, err := newSSHTarget(asset)
	if err != nil {
		return nil, err
	}

	return &authorityv1.ListAssetsResponse_Asset{
		Id:        asset.ID.String(),
		ProjectId: asset.ProjectID.String(),
		Name:      asset.Name,
		Labels:    asset.Labels,
		Target: &authorityv1.ListAssetsResponse_Asset_Ssh{Ssh: &authorityv1.ListAssetsResponse_SSHAsset{
			Target:   sshTarget,
			UserName: asset.SSH.UserName,
		}},
	}, nil
}

func newSSHSecretInput(sshLogin *authorityv1.CreateAssetRequest_SSHLogin) (service.SSHSecretInput, error) {
	switch value := sshLogin.GetSecret().(type) {
	case *authorityv1.CreateAssetRequest_SSHLogin_Password:
		return service.SSHSecretInput{
			Kind:  model.SSHSecretKindPassword,
			Value: secret.NewBytes([]byte(value.Password)),
		}, nil
	case *authorityv1.CreateAssetRequest_SSHLogin_PrivateKey:
		return service.SSHSecretInput{
			Kind:  model.SSHSecretKindPrivateKey,
			Value: secret.NewBytes([]byte(value.PrivateKey)),
		}, nil
	default:
		return service.SSHSecretInput{}, connect.NewError(connect.CodeInvalidArgument,
			errors.New("an SSH asset needs a password or a private key"))
	}
}

func newSSHTarget(asset *model.Asset) (*authorityv1.SSHTarget, error) {
	if asset.SSH == nil {
		return nil, fmt.Errorf("asset %s has no target", asset.ID)
	}

	return &authorityv1.SSHTarget{
		Host:    asset.SSH.Host,
		Port:    uint32(asset.SSH.Port),
		HostKey: asset.SSH.HostKey,
	}, nil
}
