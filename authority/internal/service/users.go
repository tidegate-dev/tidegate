package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

var (
	// ErrUserExists means the initial admin can't be created, because a user
	// exists.
	ErrUserExists = errors.New("a user already exists")
	// ErrInvalidName means a user name is empty, starts or ends with
	// whitespace, or has NUL.
	ErrInvalidName = errors.New("the name must not be empty, start or end with whitespace, or contain NUL")
)

// initialAdminLockID is the Postgres advisory lock that CreateInitialAdmin
// takes. goose uses its own lock ID for migrations.
const initialAdminLockID int64 = 0x7469646567617465 // "tidegate" in ASCII

// ValidateName returns ErrInvalidName if name is empty, starts or ends with
// whitespace, or has NUL, which Postgres refuses in text.
func ValidateName(name string) error {
	if name == "" || strings.TrimSpace(name) != name || strings.ContainsRune(name, 0) {
		return ErrInvalidName
	}

	return nil
}

// AnyExist reports whether the database holds a user.
func AnyExist(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	exists, err := sqlc.New(pool).AnyUserExists(ctx)
	if err != nil {
		return false, fmt.Errorf("checking for users: %w", err)
	}

	return exists, nil
}

// CreateInitialAdmin creates the first user, with a local password hashed
// with a salt from rnd. It returns ErrUserExists if any user exists, also when
// another run creates one at the same time.
func CreateInitialAdmin(ctx context.Context, pool *pgxpool.Pool, rnd random.Service, name string, plaintext secret.String) error {
	err := ValidateName(name)
	if err != nil {
		return err
	}

	hash, err := password.Hash(rnd, plaintext)
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
