package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

const tokenBytes = 32

// LoginTokens issues the tokens that signed-in users send with API calls.
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

	token := base64.RawURLEncoding.EncodeToString(s.rnd.Bytes(tokenBytes))
	hash := sha256.Sum256([]byte(token))

	err = s.queries.CreateLoginToken(ctx, sqlc.CreateLoginTokenParams{
		TokenHash: hash[:],
		UserID:    userID,
		Lifetime:  pgtype.Interval{Microseconds: s.lifetime.Microseconds(), Valid: true},
	})
	if err != nil {
		return secret.String{}, fmt.Errorf("storing the login token: %w", err)
	}

	return secret.New(token), nil
}
