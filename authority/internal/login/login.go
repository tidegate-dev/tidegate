// Package login implements the LoginService, which signs users in to the
// authority with a local password.
package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/semaphore"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

// maxConcurrentChecks bounds the memory that password checks take. Each
// check needs at most the memory that password.Hash uses, 64 MiB.
const maxConcurrentChecks = 4

// maxCheckWait bounds how long a login waits for a free password check, so
// clients that hold their connections open can't queue logins without end.
const maxCheckWait = 5 * time.Second

// MaxRequestBytes bounds a LoginRequest. It fits a user name and a password
// of password.MaxBytes with room to spare.
const MaxRequestBytes = 4096

const tokenBytes = 32

// The reasons a login fails. Only the log tells them apart. Clients get the
// same error for all three, so they can't tell which user names exist.
var (
	errUnknownUser   = errors.New("unknown user")
	errNoPassword    = errors.New("no local password")
	errWrongPassword = errors.New("wrong password")
)

var errBusy = errors.New("too many logins at once, try again")

// Service checks local passwords and hands out login tokens.
type Service struct {
	queries  *sqlc.Queries
	lifetime time.Duration
	checks   *semaphore.Weighted
	// checkWait is maxCheckWait, except in tests.
	checkWait time.Duration
	// fixedHash is checked when the user doesn't exist or has no password,
	// so those logins take as long as one with a wrong password.
	fixedHash string
}

// NewService returns a Service that stores login tokens with queries and
// makes them valid for lifetime.
func NewService(queries *sqlc.Queries, lifetime time.Duration) (*Service, error) {
	fixedHash, err := password.Hash(rand.Text())
	if err != nil {
		return nil, fmt.Errorf("hashing the password for unknown users: %w", err)
	}

	return &Service{
		queries:   queries,
		lifetime:  lifetime,
		checks:    semaphore.NewWeighted(maxConcurrentChecks),
		checkWait: maxCheckWait,
		fixedHash: fixedHash,
	}, nil
}

// Login checks the user's local password and returns a new login token.
func (s *Service) Login(
	ctx context.Context,
	req *connect.Request[authorityv1.LoginRequest],
) (*connect.Response[authorityv1.LoginResponse], error) {
	logger := logging.FromContext(ctx).With("user", req.Msg.GetUserName())

	token, err := s.login(ctx, req.Msg.GetUserName(), req.Msg.GetPassword())
	if err != nil {
		logger.WarnContext(ctx, "login failed", "reason", err)

		return nil, clientError(err)
	}

	logger.InfoContext(ctx, "login succeeded")

	return connect.NewResponse(&authorityv1.LoginResponse{Token: token}), nil
}

func (s *Service) login(ctx context.Context, name, plaintext string) (string, error) {
	user, err := s.queries.GetUserByName(ctx, name)

	hash, failure := user.PasswordHash.String, error(nil)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		hash, failure = s.fixedHash, errUnknownUser
	case err != nil:
		return "", fmt.Errorf("looking up the user: %w", err)
	case !user.PasswordHash.Valid:
		hash, failure = s.fixedHash, errNoPassword
	}

	ok, err := s.verify(ctx, plaintext, hash)

	switch {
	case err != nil:
		return "", err
	case failure != nil:
		return "", failure
	case !ok:
		return "", errWrongPassword
	}

	return s.createToken(ctx, user.ID)
}

func clientError(err error) error {
	switch {
	case errors.Is(err, errUnknownUser), errors.Is(err, errNoPassword), errors.Is(err, errWrongPassword):
		return connect.NewError(connect.CodeUnauthenticated, errors.New("wrong user name or password"))
	case errors.Is(err, password.ErrTooLong):
		return connect.NewError(connect.CodeInvalidArgument, password.ErrTooLong)
	case errors.Is(err, errBusy):
		return connect.NewError(connect.CodeUnavailable, errBusy)
	default:
		return err
	}
}

// verify runs password.Verify once fewer than maxConcurrentChecks run. It
// returns errBusy if no check is free within checkWait.
func (s *Service) verify(ctx context.Context, plaintext, hash string) (bool, error) {
	wait, cancel := context.WithTimeout(ctx, s.checkWait)
	defer cancel()

	err := s.checks.Acquire(wait, 1)

	switch {
	case ctx.Err() != nil:
		return false, ctx.Err()
	case err != nil:
		return false, errBusy
	}

	defer s.checks.Release(1)

	ok, err := password.Verify(plaintext, hash)
	if err != nil {
		return false, fmt.Errorf("checking the password: %w", err)
	}

	return ok, nil
}

func (s *Service) createToken(ctx context.Context, userID pgtype.UUID) (string, error) {
	err := s.queries.DeleteExpiredLoginTokens(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("deleting expired login tokens: %w", err)
	}

	random := make([]byte, tokenBytes)
	_, _ = rand.Read(random) // Since Go 1.24 it crashes the program instead of failing.

	token := base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(token))

	err = s.queries.CreateLoginToken(ctx, sqlc.CreateLoginTokenParams{
		TokenHash: hash[:],
		UserID:    userID,
		Lifetime:  pgtype.Interval{Microseconds: s.lifetime.Microseconds(), Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("storing the login token: %w", err)
	}

	return token, nil
}
