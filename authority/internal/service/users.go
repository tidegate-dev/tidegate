package service

import (
	"context"
	"errors"
	"strings"

	"github.com/The127/go-random"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/password"
	"github.com/phemera-dev/phemera/internal/secret"
)

// ErrInvalidName means a user name is empty, starts or ends with whitespace,
// or has NUL.
var ErrInvalidName = errors.New("the name must not be empty, start or end with whitespace, or contain NUL")

// ValidateName returns ErrInvalidName if name is empty, starts or ends with
// whitespace, or has NUL, which Postgres refuses in text.
func ValidateName(name string) error {
	if name == "" || strings.TrimSpace(name) != name || strings.ContainsRune(name, 0) {
		return ErrInvalidName
	}

	return nil
}

// UserChecker checks whether a user exists.
type UserChecker interface {
	// AnyExist reports whether a user exists.
	AnyExist(ctx context.Context) (bool, error)
}

// Users creates users and reports whether any exist.
type Users struct {
	users UserChecker
	units model.UnitFactory
	rnd   random.Service
}

// NewUsers returns Users that read from users and write in units of work
// from units. They take the salts of password hashes from rnd.
func NewUsers(users UserChecker, units model.UnitFactory, rnd random.Service) *Users {
	return &Users{users: users, units: units, rnd: rnd}
}

// AnyExist reports whether a user exists.
func (u *Users) AnyExist(ctx context.Context) (bool, error) {
	return u.users.AnyExist(ctx)
}

// CreateInitialAdmin creates the first user, with a local password. It
// returns model.ErrUsersExist if any user exists, also when another run
// creates one at the same time.
func (u *Users) CreateInitialAdmin(ctx context.Context, name string, plaintext secret.String) error {
	err := ValidateName(name)
	if err != nil {
		return err
	}

	hash, err := password.Hash(u.rnd, plaintext)
	if err != nil {
		return err
	}

	id, err := model.NewUserID()
	if err != nil {
		return err
	}

	passwordHash := secret.New(hash)

	unit := u.units.NewUnit()
	unit.Users().AddIfNoneExist(&model.User{ID: id, Name: name, PasswordHash: &passwordHash})

	return unit.Commit(ctx)
}
