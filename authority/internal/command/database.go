package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
)

// openDatabase connects to the database at url. It refuses a schema that's
// older than the authority expects.
func openDatabase(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := db.Open(ctx, url)
	if err != nil {
		return nil, err
	}

	err = db.CheckSchema(ctx, pool)
	if err != nil {
		pool.Close()

		if errors.Is(err, db.ErrSchemaTooOld) {
			return nil, fmt.Errorf("%w. Run authority migrate first", err)
		}

		return nil, err
	}

	return pool, nil
}
