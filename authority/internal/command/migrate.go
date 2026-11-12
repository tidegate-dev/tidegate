package command

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
)

func migrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "apply the database migrations, then exit",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := setUp(cmd)
			if err != nil {
				return err
			}

			return db.MigrateAt(ctx, cfg.Database.URL)
		},
	}
}
