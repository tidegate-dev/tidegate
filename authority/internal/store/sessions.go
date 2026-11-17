package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// SessionReader queries the sessions table on the connection pool.
type SessionReader struct {
	queries *sqlc.Queries
}

// ActiveByTokenHash returns the session with the token hash that expires
// after now, or model.ErrSessionNotFound.
func (s *SessionReader) ActiveByTokenHash(ctx context.Context, tokenHash secret.Bytes, now time.Time) (*model.Session, error) {
	row, err := s.queries.GetActiveSessionByTokenHash(ctx, sqlc.GetActiveSessionByTokenHashParams{
		TokenHash: tokenHash.Reveal(),
		Now:       databaseTime(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrSessionNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("reading the session: %w", err)
	}

	return &model.Session{
		ID:        model.SessionID(row.ID.Bytes),
		UserID:    model.UserID(row.UserID.Bytes),
		TokenHash: secret.NewBytes(row.TokenHash),
		CreatedAt: row.CreatedAt.Time,
		ExpiresAt: row.ExpiresAt.Time,
		Version:   row.Xmin.Uint32,
	}, nil
}

// SessionReadWriter belongs to a unit of work. It queries like SessionReader and
// records writes, which the unit runs at Commit.
type SessionReadWriter struct {
	SessionReader

	unit *Unit
}

// Add records the insert of session. Commit returns model.ErrConflict if
// another session has the token hash, and model.ErrUserNotFound if the user
// of the session is gone.
func (s *SessionReadWriter) Add(session *model.Session) {
	s.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		version, err := queries.InsertSession(ctx, sqlc.InsertSessionParams{
			ID:        databaseID(session.ID),
			UserID:    databaseID(session.UserID),
			TokenHash: session.TokenHash.Reveal(),
			CreatedAt: databaseTime(session.CreatedAt),
			ExpiresAt: databaseTime(session.ExpiresAt),
		})

		switch {
		case isUniqueViolation(err):
			return model.ErrConflict
		case isForeignKeyViolation(err):
			return model.ErrUserNotFound
		case err != nil:
			return fmt.Errorf("creating the session: %w", err)
		}

		session.Version = version.Uint32

		return nil
	})
}

// Remove records the delete of session, whatever its version is at Commit. A
// row that is already gone counts as removed.
func (s *SessionReadWriter) Remove(session *model.Session) {
	s.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		err := queries.DeleteSession(ctx, databaseID(session.ID))
		if err != nil {
			return fmt.Errorf("deleting the session: %w", err)
		}

		return nil
	})
}

// RemoveOfUser records the delete of every session of the user, whatever
// their versions are at Commit.
func (s *SessionReadWriter) RemoveOfUser(userID model.UserID) {
	s.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		err := queries.DeleteSessionsOfUser(ctx, databaseID(userID))
		if err != nil {
			return fmt.Errorf("deleting the user's sessions: %w", err)
		}

		return nil
	})
}

// RemoveExpiredOfUser records the delete of every session of the user that
// expires at now or earlier, whatever their versions are at Commit.
func (s *SessionReadWriter) RemoveExpiredOfUser(userID model.UserID, now time.Time) {
	s.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		err := queries.DeleteExpiredSessionsOfUser(ctx, sqlc.DeleteExpiredSessionsOfUserParams{
			UserID: databaseID(userID),
			Now:    databaseTime(now),
		})
		if err != nil {
			return fmt.Errorf("deleting the user's expired sessions: %w", err)
		}

		return nil
	})
}
