// Package users manages the authority's users.
package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
)

var (
	// ErrUserExists means the initial admin can't be created, because a user
	// exists.
	ErrUserExists = errors.New("a user already exists")
	// ErrInvalidName means a user name is empty or starts or ends with
	// whitespace.
	ErrInvalidName = errors.New("the name must not be empty or start or end with whitespace")
)

// initialAdminLockID is the Postgres advisory lock that CreateInitialAdmin
// takes. goose uses its own lock ID for migrations.
const initialAdminLockID int64 = 0x7469646567617465 // "tidegate" in ASCII

// AnyExist reports whether the database holds a user.
func AnyExist(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	exists, err := sqlc.New(pool).AnyUserExists(ctx)
	if err != nil {
		return false, fmt.Errorf("checking for users: %w", err)
	}

	return exists, nil
}

// CreateInitialAdmin creates the first user, with a local password. It
// returns ErrUserExists if any user exists, also when another run creates
// one at the same time.
func CreateInitialAdmin(ctx context.Context, pool *pgxpool.Pool, name, plaintext string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return ErrInvalidName
	}

	hash, err := password.Hash(plaintext)
	if err != nil {
		return err
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		queries := sqlc.New(tx)

		err := queries.LockTransaction(ctx, initialAdminLockID)
		if err != nil {
			return fmt.Errorf("waiting for other runs that create the initial admin: %w", err)
		}

		exists, err := queries.AnyUserExists(ctx)
		if err != nil {
			return fmt.Errorf("checking for users: %w", err)
		}

		if exists {
			return ErrUserExists
		}

		err = queries.CreateUser(ctx, sqlc.CreateUserParams{
			Name:         name,
			PasswordHash: pgtype.Text{String: hash, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating the user: %w", err)
		}

		return nil
	})
}
