package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/semaphore"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// maxConcurrentChecks bounds memory, because each check needs up to 64 MiB.
const maxConcurrentChecks = 4

// maxCheckWait keeps clients that hold connections open from queueing logins.
const maxCheckWait = 5 * time.Second

var (
	// ErrWrongCredentials comes wrapped with a reason that only the log may show.
	ErrWrongCredentials = errors.New("wrong user name or password")
	// ErrBusy means no password check became free in time.
	ErrBusy = errors.New("too many logins at once, try again")
)

// Authenticator checks users' local passwords.
type Authenticator struct {
	queries   *sqlc.Queries
	checks    *semaphore.Weighted
	checkWait time.Duration
	// fixedHash makes unknown users take as long as a wrong password.
	fixedHash string
}

// NewAuthenticator returns an Authenticator for the users in pool.
func NewAuthenticator(pool *pgxpool.Pool) (*Authenticator, error) {
	fixedHash, err := password.Hash(secret.New(rand.Text()))
	if err != nil {
		return nil, fmt.Errorf("hashing the password for unknown users: %w", err)
	}

	return &Authenticator{
		queries:   sqlc.New(pool),
		checks:    semaphore.NewWeighted(maxConcurrentChecks),
		checkWait: maxCheckWait,
		fixedHash: fixedHash,
	}, nil
}

// Authenticate returns the user's ID if plaintext is their local password.
func (a *Authenticator) Authenticate(ctx context.Context, name string, plaintext secret.String) (pgtype.UUID, error) {
	// Checked before queueing for a password check, so it never ends in ErrBusy.
	err := password.CheckMaxBytes(plaintext)
	if err != nil {
		return pgtype.UUID{}, err
	}

	user, err := a.findUser(ctx, name)

	hash, failure := user.PasswordHash.String, error(nil)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		hash, failure = a.fixedHash, fmt.Errorf("%w: unknown user", ErrWrongCredentials)
	case err != nil:
		return pgtype.UUID{}, fmt.Errorf("looking up the user: %w", err)
	case !user.PasswordHash.Valid:
		hash, failure = a.fixedHash, fmt.Errorf("%w: no local password", ErrWrongCredentials)
	}

	ok, err := a.verify(ctx, plaintext, hash)

	switch {
	case err != nil:
		return pgtype.UUID{}, err
	case failure != nil:
		return pgtype.UUID{}, failure
	case !ok:
		return pgtype.UUID{}, fmt.Errorf("%w: wrong password", ErrWrongCredentials)
	}

	return user.ID, nil
}

// findUser returns pgx.ErrNoRows for a name that no user can have, because
// Postgres refuses some of those names in a query.
func (a *Authenticator) findUser(ctx context.Context, name string) (sqlc.GetUserByNameRow, error) {
	if ValidateName(name) != nil {
		return sqlc.GetUserByNameRow{}, pgx.ErrNoRows
	}

	return a.queries.GetUserByName(ctx, name)
}

func (a *Authenticator) verify(ctx context.Context, plaintext secret.String, hash string) (bool, error) {
	wait, cancel := context.WithTimeout(ctx, a.checkWait)
	defer cancel()

	err := a.checks.Acquire(wait, 1)
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		return false, ErrBusy
	}

	defer a.checks.Release(1)

	ok, err := password.Verify(plaintext, hash)
	if err != nil {
		return false, fmt.Errorf("checking the password: %w", err)
	}

	return ok, nil
}
