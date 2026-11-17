package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/The127/go-clock"
	"github.com/The127/go-random"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

const tokenBytes = 32

// SessionFinder finds a session by the hash of its token.
type SessionFinder interface {
	// ActiveByTokenHash returns the session with the token hash that expires
	// after now, or model.ErrSessionNotFound.
	ActiveByTokenHash(ctx context.Context, tokenHash secret.Bytes, now time.Time) (*model.Session, error)
}

// Sessions issues, looks up and revokes the tokens that signed-in users
// send with API calls.
type Sessions struct {
	sessions SessionFinder
	units    model.UnitFactory
	rnd      random.Service
	clk      clock.Service
	lifetime time.Duration
}

// NewSessions returns Sessions that last lifetime from the time on clk, with
// their bytes from rnd. It reads sessions from sessions and writes them in
// units of work from units.
func NewSessions(
	sessions SessionFinder,
	units model.UnitFactory,
	rnd random.Service,
	clk clock.Service,
	lifetime time.Duration,
) *Sessions {
	return &Sessions{sessions: sessions, units: units, rnd: rnd, clk: clk, lifetime: lifetime}
}

// Issue returns a new token for the user and deletes their expired ones.
func (s *Sessions) Issue(ctx context.Context, userID model.UserID) (secret.String, error) {
	id, err := model.NewSessionID()
	if err != nil {
		return secret.String{}, err
	}

	token := secret.New(base64.RawURLEncoding.EncodeToString(s.rnd.Bytes(tokenBytes)))
	now := s.clk.Now()

	unit := s.units.NewUnit()
	unit.Sessions().RemoveExpiredOfUser(userID, now)
	unit.Sessions().Add(&model.Session{
		ID:        id,
		UserID:    userID,
		TokenHash: hash(token),
		CreatedAt: now,
		ExpiresAt: now.Add(s.lifetime),
	})

	err = unit.Commit(ctx)
	if err != nil {
		return secret.String{}, fmt.Errorf("storing the session token: %w", err)
	}

	return token, nil
}

// Session returns the session of the token. It returns ErrNotSignedIn if the
// token is unknown or expired.
func (s *Sessions) Session(ctx context.Context, token secret.String) (model.Session, error) {
	session, err := s.sessions.ActiveByTokenHash(ctx, hash(token), s.clk.Now())
	if errors.Is(err, model.ErrSessionNotFound) {
		return model.Session{}, fmt.Errorf("%w: unknown or expired token", ErrNotSignedIn)
	}

	if err != nil {
		return model.Session{}, fmt.Errorf("looking up the session token: %w", err)
	}

	return *session, nil
}

// Revoke deletes the token of the session.
func (s *Sessions) Revoke(ctx context.Context, session model.Session) error {
	unit := s.units.NewUnit()
	unit.Sessions().Remove(&session)

	err := unit.Commit(ctx)
	if err != nil {
		return fmt.Errorf("revoking a session token: %w", err)
	}

	return nil
}

// RevokeAll deletes every token of the user.
func (s *Sessions) RevokeAll(ctx context.Context, userID model.UserID) error {
	unit := s.units.NewUnit()
	unit.Sessions().RemoveOfUser(userID)

	err := unit.Commit(ctx)
	if err != nil {
		return fmt.Errorf("revoking a user's session tokens: %w", err)
	}

	return nil
}

func hash(token secret.String) secret.Bytes {
	sum := sha256.Sum256([]byte(token.Reveal()))

	return secret.NewBytes(sum[:])
}
