// Package secretengine has the interface of a store for secret values, like
// the password or the private key for an SSH host.
package secretengine

import (
	"context"
	"errors"

	"github.com/phemera-dev/phemera/internal/secret"
)

// ErrNotFound means that an engine holds no value under a reference.
var ErrNotFound = errors.New("secret not found")

// Reference names one stored value within the engine that returned it.
type Reference string

// Engine stores secret values and returns them by reference. It doesn't look
// into a value.
type Engine interface {
	// Store saves the value and returns the reference to fetch it with.
	Store(ctx context.Context, value secret.Bytes) (Reference, error)

	// Fetch returns the value that was stored under the reference. If there
	// is none, the error wraps ErrNotFound.
	Fetch(ctx context.Context, reference Reference) (secret.Bytes, error)

	// Delete removes the value that was stored under the reference. If there
	// is none, the error wraps ErrNotFound.
	Delete(ctx context.Context, reference Reference) error
}
