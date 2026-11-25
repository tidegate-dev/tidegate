// Package command defines the authority command and its subcommands.
package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/phemera-dev/phemera/authority/internal/config"
	"github.com/phemera-dev/phemera/authority/internal/db"
	"github.com/phemera-dev/phemera/authority/internal/server"
	"github.com/phemera-dev/phemera/internal/logging"
)

// ErrInvalidConfig means the config couldn't be loaded. No logger is set up
// then.
var ErrInvalidConfig = errors.New("invalid config")

// New returns the authority command with all its subcommands.
func New() *cli.Command {
	return &cli.Command{
		Name:  "authority",
		Usage: "run the phemera authority",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := setUp(cmd)
			if err != nil {
				return err
			}

			return withMigrateHint(server.Run(ctx, cfg))
		},
		Commands: []*cli.Command{
			migrateCommand(),
			createInitialAdminCommand(),
		},
	}
}

// setUp rejects arguments, loads the config and sets up logging from it.
// The commands call it themselves, so --help works without a config.
func setUp(cmd *cli.Command) (config.Config, error) {
	if cmd.Args().Present() {
		return config.Config{}, fmt.Errorf("unknown command %q", cmd.Args().First())
	}

	return loadConfig()
}

// loadConfig loads the config and sets up logging from it.
func loadConfig() (config.Config, error) {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		return config.Config{}, fmt.Errorf("%w:\n%w", ErrInvalidConfig, err)
	}

	slog.SetDefault(logging.New(os.Stderr, cfg.Log.Level, cfg.Log.Format))
	slog.Debug("loaded config",
		"external_api_listen_address", cfg.ExternalAPI.ListenAddress,
		"internal_api_listen_address", cfg.InternalAPI.ListenAddress)

	return cfg, nil
}

func withMigrateHint(err error) error {
	if errors.Is(err, db.ErrSchemaTooOld) {
		return fmt.Errorf("%w. Run authority migrate first", err)
	}

	return err
}
