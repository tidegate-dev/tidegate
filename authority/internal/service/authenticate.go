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
)

// maxConcurrentChecks bounds the memory that password checks take. Each
// check needs at most the memory that password.Hash uses, 64 MiB.
const maxConcurrentChecks = 4

// maxCheckWait bounds how long Authenticate waits for a free password check,
// so clients that hold their connections open can't queue logins without
// end.
const maxCheckWait = 5 * time.Second

var (
	// ErrWrongCredentials means the user doesn't exist, has no local password
	// or gave another password. Authenticate wraps it with the reason, for the
	// log. Clients should only learn that the credentials are wrong, so they
	// can't tell which user names exist.
	ErrWrongCredentials = errors.New("wrong user name or password")
	// ErrBusy means no password check became free in time.
	ErrBusy = errors.New("too many logins at once, try again")
)

// Authenticator checks users' local passwords.
type Authenticator struct {
	queries *sqlc.Queries
	checks  *semaphore.Weighted
	// checkWait is maxCheckWait, except in tests.
	checkWait time.Duration
	// fixedHash is checked when the user doesn't exist or has no password,
	// so those take as long as a wrong password.
	fixedHash string
}

// NewAuthenticator returns an Authenticator for the users in pool.
func NewAuthenticator(pool *pgxpool.Pool) (*Authenticator, error) {
	fixedHash, err := password.Hash(rand.Text())
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

// Authenticate returns the ID of the user called name if plaintext is their
// local password. The name ignores letter case. It returns
// ErrWrongCredentials if not, password.ErrTooLong for a password over
// password.MaxBytes, and ErrBusy if no password check becomes free in time.
func (a *Authenticator) Authenticate(ctx context.Context, name, plaintext string) (pgtype.UUID, error) {
	user, err := a.queries.GetUserByName(ctx, name)

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

func (a *Authenticator) verify(ctx context.Context, plaintext, hash string) (bool, error) {
	wait, cancel := context.WithTimeout(ctx, a.checkWait)
	defer cancel()

	err := a.checks.Acquire(wait, 1)

	switch {
	case ctx.Err() != nil:
		return false, ctx.Err()
	case err != nil:
		return false, ErrBusy
	}

	defer a.checks.Release(1)

	ok, err := password.Verify(plaintext, hash)
	if err != nil {
		return false, fmt.Errorf("checking the password: %w", err)
	}

	return ok, nil
}
