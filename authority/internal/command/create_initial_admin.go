package command

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/The127/go-random"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

func createInitialAdminCommand() *cli.Command {
	return &cli.Command{
		Name:      "create-initial-admin",
		Usage:     "create the first user with a local password, then exit",
		ArgsUsage: "<name>",
		Description: "Asks for the password twice on a terminal, or reads it from the first line of stdin.\n" +
			"Works only while the database holds no user.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return errors.New("create-initial-admin takes exactly one argument, the user's name")
			}

			err := service.ValidateName(cmd.Args().First())
			if err != nil {
				return err
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			return withMigrateHint(createInitialAdmin(ctx, cfg.Database, cmd.Args().First(), cmd.Reader, cmd.ErrWriter))
		},
	}
}

// createInitialAdmin checks the schema and looks for users before it asks
// for the password, so the user doesn't type it for nothing.
func createInitialAdmin(ctx context.Context, cfg config.Database, name string, in io.Reader, out io.Writer) error {
	pool, err := db.OpenMigrated(ctx, cfg.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	exists, err := service.AnyExist(ctx, pool)
	if err != nil {
		return err
	}

	if exists {
		return service.ErrUserExists
	}

	plaintext, err := readPassword(ctx, terminal.New(in, out))
	if err != nil {
		return err
	}

	err = service.CreateInitialAdmin(ctx, pool, random.NewCryptoRandom(), name, plaintext)
	if err != nil {
		return err
	}

	slog.Info("created the initial admin", "name", name)

	return nil
}
