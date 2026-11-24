package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

// IANA assigns port 22 to SSH.
const defaultSSHPort = 22

// The authority accepts a private key of at most this many bytes.
const maxPrivateKeyBytes = 16384

var errAddress = errors.New("want an address like host or host:port")

type sshTargetResult struct {
	Host     string `json:"host"     yaml:"host"`
	Port     uint32 `json:"port"     yaml:"port"`
	HostKey  string `json:"hostKey"  yaml:"hostKey"`
	UserName string `json:"userName" yaml:"userName"`
}

type assetResult struct {
	ID        string            `json:"id"            yaml:"id"`
	ProjectID string            `json:"projectId"     yaml:"projectId"`
	Name      string            `json:"name"          yaml:"name"`
	Labels    map[string]string `json:"labels"        yaml:"labels"`
	SSH       *sshTargetResult  `json:"ssh,omitempty" yaml:"ssh,omitempty"`
}

func newSSHTargetResult(target *authorityv1.SSHTarget) *sshTargetResult {
	if target == nil {
		return nil
	}

	return &sshTargetResult{
		Host:     target.GetHost(),
		Port:     target.GetPort(),
		HostKey:  target.GetHostKey(),
		UserName: target.GetUserName(),
	}
}

func newAssetResult(response *authorityv1.CreateAssetResponse) assetResult {
	result := assetResult{
		ID:        response.GetId(),
		ProjectID: response.GetProjectId(),
		Name:      response.GetName(),
		Labels:    map[string]string{},
		SSH:       newSSHTargetResult(response.GetSsh()),
	}

	maps.Copy(result.Labels, response.GetLabels())

	return result
}

func (result assetResult) String() string {
	return quoteIfUnprintable(result.ID) + "\t" + quoteIfUnprintable(result.Name)
}

func (result assetResult) tableRow() []string {
	return []string{
		quoteIfUnprintable(result.ID),
		quoteIfUnprintable(result.Name),
		quoteIfUnprintable(result.address()),
		quoteIfUnprintable(result.labelsText()),
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

func newAssetsResult(assets []*authorityv1.ListAssetsResponse_Asset) assetsResult {
	result := assetsResult{Assets: make([]assetResult, 0, len(assets))}

	for _, asset := range assets {
		listed := assetResult{
			ID:        asset.GetId(),
			ProjectID: asset.GetProjectId(),
			Name:      asset.GetName(),
			Labels:    map[string]string{},
			SSH:       newSSHTargetResult(asset.GetSsh()),
		}

		maps.Copy(listed.Labels, asset.GetLabels())

		result.Assets = append(result.Assets, listed)
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

	return tableText([]string{"ID", "NAME", "ADDRESS", "LABELS"}, rows)
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
						Usage:     "add an SSH host to a project and print the asset's ID and name",
						ArgsUsage: "<name>",
						// urfave/cli splits the value of a repeated flag at commas by default.
						DisableSliceFlagSeparator: true,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project-name",
								Usage:    "name of the project that gets the asset",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "address",
								Usage:    "the host's DNS name or IP address, as host or host:port, with port 22 by default",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "host-key",
								Usage:    "the host's public key, like the content of /etc/ssh/ssh_host_ed25519_key.pub",
								Required: true,
							},
							&cli.StringSliceFlag{
								Name:  "label",
								Usage: "a label as key=value that roles select the asset by, can be given several times",
							},
							&cli.StringFlag{
								Name:     "user-name",
								Usage:    "the user that tidegate logs in as on the host",
								Required: true,
							},
							&cli.BoolFlag{
								Name: "password",
								Usage: "the user logs in with a password. The command asks for it on a terminal, " +
									"or reads it from the first line of stdin. The command needs this flag or --private-key-file",
							},
							&cli.StringFlag{
								Name: "private-key-file",
								Usage: "the user logs in with the private key in this file, which has no passphrase. " +
									"The command needs this flag or --password",
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
				Usage: "print the ID, the name, the address and the labels of every asset of a project",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "project-name",
						Usage:    "name of the project whose assets are listed",
						Required: true,
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

	privateKeyFile := cmd.String("private-key-file")

	err = checkSSHSecretFlags(cmd.Bool("password"), privateKeyFile)
	if err != nil {
		return err
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	asset := &authorityv1.CreateAssetRequest_SSHAsset{Target: &authorityv1.SSHTarget{
		Host:     host,
		Port:     port,
		HostKey:  hostKey,
		UserName: cmd.String("user-name"),
	}}

	err = setSSHSecret(ctx, asset, privateKeyFile, cmd.Root().Reader, cmd.Root().ErrWriter)
	if err != nil {
		return err
	}

	response, err := clients.Asset().CreateAsset(ctx, connect.NewRequest(&authorityv1.CreateAssetRequest{
		ProjectName: cmd.String("project-name"),
		Name:        cmd.Args().First(),
		Labels:      labels,
		Target:      &authorityv1.CreateAssetRequest_Ssh{Ssh: asset},
	}))
	if err != nil {
		return callFailed("creating the asset", connection, err)
	}

	return printResult(cmd, newAssetResult(response.Msg))
}

func checkSSHSecretFlags(askForPassword bool, privateKeyFile string) error {
	switch {
	case askForPassword && privateKeyFile != "":
		return errors.New("--password and --private-key-file can't be combined")
	case !askForPassword && privateKeyFile == "":
		return errors.New("want --password or --private-key-file")
	default:
		return nil
	}
}

func setSSHSecret(
	ctx context.Context,
	asset *authorityv1.CreateAssetRequest_SSHAsset,
	privateKeyFile string,
	in io.Reader,
	prompts io.Writer,
) error {
	if privateKeyFile != "" {
		privateKey, err := readPrivateKeyFile(privateKeyFile)
		if err != nil {
			return fmt.Errorf("--private-key-file: %w", err)
		}

		asset.Secret = &authorityv1.CreateAssetRequest_SSHAsset_PrivateKey{PrivateKey: privateKey}

		return nil
	}

	prompt := "Password of " + quoteIfUnprintable(asset.GetTarget().GetUserName()) + " on the host: "

	password, err := readPassword(ctx, terminal.New(in, prompts), prompt)
	if err != nil {
		return err
	}

	asset.Secret = &authorityv1.CreateAssetRequest_SSHAsset_Password{Password: password}

	return nil
}

func readPrivateKeyFile(path string) (string, error) {
	file, err := os.Open(path) //nolint:gosec // The path is the file that the user names.
	if err != nil {
		return "", err
	}

	defer func() {
		_ = file.Close()
	}()

	// A path like /dev/zero never ends.
	privateKey, err := io.ReadAll(io.LimitReader(file, maxPrivateKeyBytes+1))
	if err != nil {
		return "", err
	}

	if len(privateKey) > maxPrivateKeyBytes {
		return "", fmt.Errorf("the file has more than %d bytes, which is too big for a private key", maxPrivateKeyBytes)
	}

	return string(privateKey), nil
}

func assetList(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 0 {
		return errors.New("want no arguments")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Asset().ListAssets(ctx, connect.NewRequest(&authorityv1.ListAssetsRequest{
		ProjectName: cmd.String("project-name"),
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
	lines := strings.FieldsFunc(text, func(character rune) bool { return character == '\n' || character == '\r' })
	lines = slices.DeleteFunc(lines, func(line string) bool { return strings.TrimSpace(line) == "" })

	if len(lines) > 1 {
		return "", errors.New("want one host key, not several lines")
	}

	fields := strings.Fields(strings.Join(lines, ""))
	if len(fields) < 2 {
		return "", errors.New(`want a host key like "ssh-ed25519 AAAA..."`)
	}

	return fields[0] + " " + fields[1], nil
}
