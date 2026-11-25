package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/phemera-dev/phemera/authority/internal/db/sqlc"
	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/internal/secret"
)

// goose takes a different advisory lock ID for migrations.
const usersEmptyLockID int64 = 0x7068656d657261 // "phemera" in ASCII

// UserReader queries the users table on the connection pool.
type UserReader struct {
	queries *sqlc.Queries
}

// FindByName returns the user with the name, ignoring letter case, or
// model.ErrUserNotFound.
func (u *UserReader) FindByName(ctx context.Context, name string) (*model.User, error) {
	row, err := u.queries.GetUserByName(ctx, name)
	// Postgres refuses text with NUL or invalid UTF-8 with this code.
	if errors.Is(err, pgx.ErrNoRows) || hasCode(err, pgerrcode.CharacterNotInRepertoire) {
		return nil, model.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("reading the user: %w", err)
	}

	user := &model.User{ID: model.UserID(row.ID.Bytes), Name: row.Name, Version: row.Xmin.Uint32}

	if row.PasswordHash.Valid {
		hash := secret.New(row.PasswordHash.String)
		user.PasswordHash = &hash
	}

	return user, nil
}

// AnyExist reports whether the table holds a user.
func (u *UserReader) AnyExist(ctx context.Context) (bool, error) {
	exists, err := u.queries.AnyUserExists(ctx)
	if err != nil {
		return false, fmt.Errorf("checking for users: %w", err)
	}

	return exists, nil
}

// UserReadWriter belongs to a unit of work. It queries like UserReader and records
// writes, which the unit runs at Commit.
type UserReadWriter struct {
	UserReader

	unit *Unit
}

// Add records the insert of user. Commit returns model.ErrUserNameTaken if
// another user has the name.
func (u *UserReadWriter) Add(user *model.User) {
	u.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		return insertUser(ctx, queries, user)
	})
}

// AddIfNoneExist records the insert of user into an empty table. Commit
// returns model.ErrUsersExist if the table holds a user by then.
func (u *UserReadWriter) AddIfNoneExist(user *model.User) {
	u.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		// Postgres can't lock rows that don't exist yet.
		err := queries.LockTransaction(ctx, usersEmptyLockID)
		if err != nil {
			return fmt.Errorf("waiting for the lock on the empty users table: %w", err)
		}

		exists, err := queries.AnyUserExists(ctx)
		if err != nil {
			return fmt.Errorf("checking for users: %w", err)
		}

		if exists {
			return model.ErrUsersExist
		}

		return insertUser(ctx, queries, user)
	})
}

// Update records the update of every column of user. Commit returns
// model.ErrChangedSinceRead if the row changed since user was read,
// model.ErrUserNameTaken if another user has the name, and
// model.ErrUserNotFound if it's gone.
func (u *UserReadWriter) Update(user *model.User) {
	u.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		result, err := queries.UpdateUser(ctx, sqlc.UpdateUserParams{
			ID:           databaseID(user.ID),
			Name:         pgtype.Text{String: user.Name, Valid: true},
			PasswordHash: databaseHash(user.PasswordHash),
			Version:      databaseVersion(user.Version),
		})

		switch {
		case isUniqueViolationOf(err, "users_name_unique"):
			return model.ErrUserNameTaken
		case err != nil:
			return fmt.Errorf("updating the user: %w", err)
		case result.Version.Valid:
			user.Version = result.Version.Uint32
			return nil
		case result.Found.Bool:
			return model.ErrChangedSinceRead
		default:
			return model.ErrUserNotFound
		}
	})
}

// Remove records the delete of user. Commit returns
// model.ErrChangedSinceRead if the row changed since user was read. A row
// that is already gone counts as removed.
func (u *UserReadWriter) Remove(user *model.User) {
	u.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		result, err := queries.DeleteUser(ctx, sqlc.DeleteUserParams{
			ID:      databaseID(user.ID),
			Version: databaseVersion(user.Version),
		})
		if err != nil {
			return fmt.Errorf("deleting the user: %w", err)
		}

		if !result.Deleted && result.Found {
			return model.ErrChangedSinceRead
		}

		return nil
	})
}

func insertUser(ctx context.Context, queries *sqlc.Queries, user *model.User) error {
	version, err := queries.InsertUser(ctx, sqlc.InsertUserParams{
		ID:           databaseID(user.ID),
		Name:         user.Name,
		PasswordHash: databaseHash(user.PasswordHash),
	})
	if isUniqueViolationOf(err, "users_name_unique") {
		return model.ErrUserNameTaken
	}

	if err != nil {
		return fmt.Errorf("creating the user: %w", err)
	}

	user.Version = version.Uint32

	return nil
}
