package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
)

const tokenBytes = 32

// LoginTokens hands out the tokens that signed-in users send with their
// API calls.
type LoginTokens struct {
	queries  *sqlc.Queries
	lifetime time.Duration
}

// NewLoginTokens returns LoginTokens that keeps tokens in pool and makes
// them valid for lifetime.
func NewLoginTokens(pool *pgxpool.Pool, lifetime time.Duration) *LoginTokens {
	return &LoginTokens{queries: sqlc.New(pool), lifetime: lifetime}
}

// Issue returns a new login token for the user and deletes their expired
// ones. It stores only the token's SHA-256 hash, so a copy of the database
// can't sign anyone in.
func (s *LoginTokens) Issue(ctx context.Context, userID pgtype.UUID) (string, error) {
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
