package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/secret"
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
		asset, err = s.assets.CreateSSH(ctx, req.Msg.GetProjectName(), req.Msg.GetName(), req.Msg.GetLabels(), model.SSHTarget{
			Host:    target.Ssh.GetHost(),
			Port:    uint16(target.Ssh.GetPort()), //nolint:gosec // The API definition allows 1 to 65535.
			HostKey: target.Ssh.GetHostKey(),
		}, newSSHLoginInput(req.Msg))
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
	ssh, err := sshTarget(asset)
	if err != nil {
		return nil, err
	}

	return &authorityv1.CreateAssetResponse{
		Id:        asset.ID.String(),
		ProjectId: asset.ProjectID.String(),
		Name:      asset.Name,
		Labels:    asset.Labels,
		Target:    &authorityv1.CreateAssetResponse_Ssh{Ssh: ssh},
	}, nil
}

func newListedAsset(asset *model.Asset) (*authorityv1.ListAssetsResponse_Asset, error) {
	ssh, err := sshTarget(asset)
	if err != nil {
		return nil, err
	}

	return &authorityv1.ListAssetsResponse_Asset{
		Id:        asset.ID.String(),
		ProjectId: asset.ProjectID.String(),
		Name:      asset.Name,
		Labels:    asset.Labels,
		Target:    &authorityv1.ListAssetsResponse_Asset_Ssh{Ssh: ssh},
	}, nil
}

func newSSHLoginInput(msg *authorityv1.CreateAssetRequest) *service.SSHLoginInput {
	switch value := msg.GetSshSecret().(type) {
	case *authorityv1.CreateAssetRequest_SshPassword:
		return &service.SSHLoginInput{
			UserName: msg.GetSsh().GetUserName(),
			Kind:     model.SSHLoginKindPassword,
			Value:    secret.NewBytes([]byte(value.SshPassword)),
		}
	case *authorityv1.CreateAssetRequest_SshPrivateKey:
		return &service.SSHLoginInput{
			UserName: msg.GetSsh().GetUserName(),
			Kind:     model.SSHLoginKindPrivateKey,
			Value:    secret.NewBytes([]byte(value.SshPrivateKey)),
		}
	default:
		return nil
	}
}

func sshTarget(asset *model.Asset) (*authorityv1.SSHTarget, error) {
	if asset.SSH == nil {
		return nil, fmt.Errorf("asset %s has no target", asset.ID)
	}

	target := &authorityv1.SSHTarget{
		Host:    asset.SSH.Host,
		Port:    uint32(asset.SSH.Port),
		HostKey: asset.SSH.HostKey,
	}

	if asset.SSH.Login != nil {
		target.UserName = asset.SSH.Login.UserName
	}

	return target, nil
}
