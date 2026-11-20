package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/The127/go-random"
	"golang.org/x/sync/semaphore"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
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

// UserFinder finds a user by name.
type UserFinder interface {
	// ByName returns the user with the name, ignoring letter case, or
	// model.ErrUserNotFound.
	ByName(ctx context.Context, name string) (*model.User, error)
}

// Authenticator checks users' local passwords.
type Authenticator struct {
	users     UserFinder
	checks    *semaphore.Weighted
	checkWait time.Duration
	// fixedHash makes unknown users take as long as a wrong password.
	fixedHash string
}

// NewAuthenticator returns an Authenticator for users. It takes the password
// and salt of fixedHash from rnd.
func NewAuthenticator(users UserFinder, rnd random.Service) (*Authenticator, error) {
	fixedHash, err := password.Hash(rnd, secret.New(rnd.Text()))
	if err != nil {
		return nil, fmt.Errorf("hashing the password for unknown users: %w", err)
	}

	return &Authenticator{
		users:     users,
		checks:    semaphore.NewWeighted(maxConcurrentChecks),
		checkWait: maxCheckWait,
		fixedHash: fixedHash,
	}, nil
}

// Authenticate returns the user's ID if plaintext is their local password.
func (a *Authenticator) Authenticate(ctx context.Context, name string, plaintext secret.String) (model.UserID, error) {
	// Checked before queueing for a password check, so it never ends in ErrBusy.
	err := password.CheckMaxBytes(plaintext)
	if err != nil {
		return model.UserID{}, err
	}

	user, err := findUser(ctx, a.users, name)

	var (
		hash    string
		failure error
	)

	switch {
	case errors.Is(err, model.ErrUserNotFound):
		hash, failure = a.fixedHash, fmt.Errorf("%w: unknown user", ErrWrongCredentials)
	case err != nil:
		return model.UserID{}, fmt.Errorf("looking up the user: %w", err)
	case user.PasswordHash == nil:
		hash, failure = a.fixedHash, fmt.Errorf("%w: no local password", ErrWrongCredentials)
	default:
		hash = user.PasswordHash.Reveal()
	}

	ok, err := a.verify(ctx, plaintext, hash)

	switch {
	case err != nil:
		return model.UserID{}, err
	case failure != nil:
		return model.UserID{}, failure
	case !ok:
		return model.UserID{}, fmt.Errorf("%w: wrong password", ErrWrongCredentials)
	}

	return user.ID, nil
}

// findUser returns model.ErrUserNotFound for a name that no user can have,
// because Postgres refuses some of those names in a query.
func findUser(ctx context.Context, users UserFinder, name string) (*model.User, error) {
	if ValidateName(name) != nil {
		return nil, model.ErrUserNotFound
	}

	return users.ByName(ctx, name)
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
