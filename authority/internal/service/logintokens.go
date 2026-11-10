package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

const tokenBytes = 32

// Session is one login of a user, found by its login token.
type Session struct {
	ID     pgtype.UUID
	UserID pgtype.UUID
}

// LoginTokens issues, looks up and revokes the tokens that signed-in users
// send with API calls.
type LoginTokens struct {
	queries  *sqlc.Queries
	rnd      random.Service
	lifetime time.Duration
}

// NewLoginTokens returns LoginTokens that last lifetime, with their bytes
// from rnd.
func NewLoginTokens(pool *pgxpool.Pool, rnd random.Service, lifetime time.Duration) *LoginTokens {
	return &LoginTokens{queries: sqlc.New(pool), rnd: rnd, lifetime: lifetime}
}

// Issue returns a new token for the user and deletes their expired ones.
func (s *LoginTokens) Issue(ctx context.Context, userID pgtype.UUID) (secret.String, error) {
	err := s.queries.DeleteExpiredLoginTokens(ctx, userID)
	if err != nil {
		return secret.String{}, fmt.Errorf("deleting expired login tokens: %w", err)
	}

	token := secret.New(base64.RawURLEncoding.EncodeToString(s.rnd.Bytes(tokenBytes)))

	err = s.queries.CreateLoginToken(ctx, sqlc.CreateLoginTokenParams{
		TokenHash: hash(token),
		UserID:    userID,
		Lifetime:  pgtype.Interval{Microseconds: s.lifetime.Microseconds(), Valid: true},
	})
	if err != nil {
		return secret.String{}, fmt.Errorf("storing the login token: %w", err)
	}

	return token, nil
}

// Session returns the session of the token. It returns ErrNotSignedIn if the
// token is unknown or expired.
func (s *LoginTokens) Session(ctx context.Context, token secret.String) (Session, error) {
	row, err := s.queries.GetLoginSession(ctx, hash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, fmt.Errorf("%w: unknown or expired token", ErrNotSignedIn)
	}

	if err != nil {
		return Session{}, fmt.Errorf("looking up the login token: %w", err)
	}

	return Session{ID: row.ID, UserID: row.UserID}, nil
}

// Revoke deletes the token of the session.
func (s *LoginTokens) Revoke(ctx context.Context, sessionID pgtype.UUID) error {
	err := s.queries.RevokeLoginToken(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("revoking a login token: %w", err)
	}

	return nil
}

// RevokeAll deletes every token of the user.
func (s *LoginTokens) RevokeAll(ctx context.Context, userID pgtype.UUID) error {
	err := s.queries.RevokeUserLoginTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("revoking a user's login tokens: %w", err)
	}

	return nil
}

func hash(token secret.String) []byte {
	sum := sha256.Sum256([]byte(token.Reveal()))

	return sum[:]
}
