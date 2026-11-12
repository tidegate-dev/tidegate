// Package filelock lets one process at a time hold a lock file, on Linux,
// macOS and Windows.
package filelock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

const retryDelay = 20 * time.Millisecond

// Lock is a lock file that this process holds.
type Lock struct {
	lock *flock.Flock
}

// Acquire waits until nobody else holds the lock file at path, and then
// holds it. It gives up after limit. It creates the file if it doesn't exist,
// readable only by the owner.
func Acquire(path string, limit time.Duration) (*Lock, error) {
	lock := flock.New(path)

	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()

	locked, err := lock.TryLockContext(ctx, retryDelay)
	if errors.Is(err, context.DeadlineExceeded) || (err == nil && !locked) {
		return nil, fmt.Errorf("another process holds %s for more than %s", path, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("locking %s: %w", path, err)
	}

	return &Lock{lock: lock}, nil
}

// Release lets the next one acquire the lock file. The file stays in place.
func (l *Lock) Release() error {
	err := l.lock.Unlock()
	if err != nil {
		return fmt.Errorf("unlocking %s: %w", l.lock.Path(), err)
	}

	return nil
}
