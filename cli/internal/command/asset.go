package command

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// IANA assigns port 22 to SSH.
const defaultSSHPort = 22

var errAddress = errors.New("want an address like host or host:port")

type sshTargetResult struct {
	Host    string `json:"host"    yaml:"host"`
	Port    uint32 `json:"port"    yaml:"port"`
	HostKey string `json:"hostKey" yaml:"hostKey"`
}

type assetResult struct {
	ID        string            `json:"id"            yaml:"id"`
	ProjectID string            `json:"projectId"     yaml:"projectId"`
	Name      string            `json:"name"          yaml:"name"`
	Labels    map[string]string `json:"labels"        yaml:"labels"`
	SSH       *sshTargetResult  `json:"ssh,omitempty" yaml:"ssh,omitempty"`
}

func newAssetResult(asset *authorityv1.Asset) assetResult {
	result := assetResult{
		ID:        asset.GetId(),
		ProjectID: asset.GetProjectId(),
		Name:      asset.GetName(),
		Labels:    map[string]string{},
	}

	maps.Copy(result.Labels, asset.GetLabels())

	if target := asset.GetSsh(); target != nil {
		result.SSH = &sshTargetResult{Host: target.GetHost(), Port: target.GetPort(), HostKey: target.GetHostKey()}
	}

	return result
}

func (result assetResult) String() string {
	return quoteIfUnprintable(result.Name) + "\t" + quoteIfUnprintable(result.ID)
}

func (result assetResult) tableRow() []string {
	return []string{
		quoteIfUnprintable(result.Name),
		quoteIfUnprintable(result.address()),
		quoteIfUnprintable(result.labelsText()),
		quoteIfUnprintable(result.ID),
	}
}

func (result assetResult) address() string {
	if result.SSH == nil {
		return ""
	}

	return net.JoinHostPort(result.SSH.Host, strconv.FormatUint(uint64(result.SSH.Port), 10))
}

func (result assetResult) labelsText() string {
	pairs := make([]string, 0, len(result.Labels))
	for _, key := range slices.Sorted(maps.Keys(result.Labels)) {
		pairs = append(pairs, key+"="+result.Labels[key])
	}

	return strings.Join(pairs, ",")
}

type assetsResult struct {
	Assets []assetResult `json:"assets" yaml:"assets"`
}

func newAssetsResult(assets []*authorityv1.Asset) assetsResult {
	result := assetsResult{Assets: make([]assetResult, 0, len(assets))}
	for _, asset := range assets {
		result.Assets = append(result.Assets, newAssetResult(asset))
	}

	return result
}

func (result assetsResult) String() string {
	if len(result.Assets) == 0 {
		return "No assets in the project. Run `tidegate asset create ssh` to add an SSH host."
	}

	rows := make([][]string, 0, len(result.Assets))
	for _, asset := range result.Assets {
		rows = append(rows, asset.tableRow())
	}

	return tableText([]string{"NAME", "ADDRESS", "LABELS", "ID"}, rows)
}

func assetCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "asset",
		Usage: "add assets to a project and list them",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "add an asset of one kind to a project",
				Commands: []*cli.Command{
					{
						Name:      "ssh",
						Usage:     "add an SSH host to a project and print the asset's name and ID",
						ArgsUsage: "<name>",
						// urfave/cli splits the value of a repeated flag at commas by default.
						DisableSliceFlagSeparator: true,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "project",
								Usage: "name of the project that gets the asset",
							},
							&cli.StringFlag{
								Name:  "address",
								Usage: "the host's DNS name or IP address, as host or host:port, with port 22 by default",
							},
							&cli.StringFlag{
								Name:  "host-key",
								Usage: "the host's public key, like the content of /etc/ssh/ssh_host_ed25519_key.pub",
							},
							&cli.StringSliceFlag{
								Name:  "label",
								Usage: "a label as key=value that roles select the asset by, can be given several times",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return assetCreateSSH(ctx, cmd, store)
						},
					},
				},
			},
			{
				Name:  "list",
				Usage: "print the name, the address, the labels and the ID of every asset of a project",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "project",
						Usage: "name of the project whose assets are listed",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return assetList(ctx, cmd, store)
				},
			},
		},
	}
}

func assetCreateSSH(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one name")
	}

	if cmd.String("project") == "" {
		return errors.New("want a project, name it with --project")
	}

	host, port, err := parseAddress(cmd.String("address"))
	if err != nil {
		return fmt.Errorf("--address: %w", err)
	}

	hostKey, err := hostKeyWithoutComment(cmd.String("host-key"))
	if err != nil {
		return fmt.Errorf("--host-key: %w", err)
	}

	labels, err := parseLabels(cmd.StringSlice("label"))
	if err != nil {
		return fmt.Errorf("--label: %w", err)
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Asset().CreateAsset(ctx, connect.NewRequest(&authorityv1.CreateAssetRequest{
		Project: cmd.String("project"),
		Name:    cmd.Args().First(),
		Labels:  labels,
		Target:  &authorityv1.CreateAssetRequest_Ssh{Ssh: &authorityv1.SSHTarget{Host: host, Port: port, HostKey: hostKey}},
	}))
	if err != nil {
		return callFailed("creating the asset", connection, err)
	}

	return printResult(cmd, newAssetResult(response.Msg.GetAsset()))
}

func assetList(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 0 {
		return errors.New("want no arguments")
	}

	if cmd.String("project") == "" {
		return errors.New("want a project, name it with --project")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Asset().ListAssets(ctx, connect.NewRequest(&authorityv1.ListAssetsRequest{
		Project: cmd.String("project"),
	}))
	if err != nil {
		return callFailed("listing the assets", connection, err)
	}

	return printResult(cmd, newAssetsResult(response.Msg.GetAssets()))
}

func parseAddress(address string) (string, uint32, error) {
	if _, err := netip.ParseAddr(address); err == nil {
		return address, defaultSSHPort, nil
	}

	hasPort := strings.Contains(address, ":") && !strings.HasSuffix(address, "]")
	if !hasPort {
		address += ":" + strconv.Itoa(defaultSSHPort)
	}

	host, portText, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return "", 0, errAddress
	}

	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return "", 0, errAddress
	}

	return host, uint32(port), nil
}

func parseLabels(pairs []string) (map[string]string, error) {
	labels := make(map[string]string, len(pairs))

	for _, pair := range pairs {
		key, value, hasValue := strings.Cut(pair, "=")
		if !hasValue || key == "" {
			return nil, fmt.Errorf("label %q must look like key=value", pair)
		}

		if _, isGiven := labels[key]; isGiven {
			return nil, fmt.Errorf("label %q is given twice", key)
		}

		labels[key] = value
	}

	return labels, nil
}

// sshd writes a comment like "root@bastion" after the key in its .pub files.
func hostKeyWithoutComment(text string) (string, error) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return "", errors.New(`want a host key like "ssh-ed25519 AAAA..."`)
	}

	return fields[0] + " " + fields[1], nil
}
