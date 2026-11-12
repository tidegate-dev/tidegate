// Package command defines the authority command and its subcommands.
package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

const shutdownTimeout = 10 * time.Second

// ErrInvalidConfig marks errors that main prints as they are, because the
// logger is only set up from a valid config.
var ErrInvalidConfig = errors.New("invalid config")

// New returns the authority command with all its subcommands.
func New() *cli.Command {
	return &cli.Command{
		Name:  "authority",
		Usage: "run the tidegate authority",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := setUp(cmd)
			if err != nil {
				return err
			}

			return withMigrateHint(run(ctx, cfg, shutdownTimeout))
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
	slog.Debug("loaded config", "listen_address", cfg.Server.ListenAddress)

	return cfg, nil
}

func withMigrateHint(err error) error {
	if errors.Is(err, db.ErrSchemaTooOld) {
		return fmt.Errorf("%w. Run authority migrate first", err)
	}

	return err
}
