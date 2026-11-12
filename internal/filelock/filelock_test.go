package filelock_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/filelock"
)

func TestAcquireGivesUpAfterTheLimitWhileTheLockIsHeld(t *testing.T) {
	t.Parallel()

	// arrange
	path := filepath.Join(t.TempDir(), "lock")

	held, err := filelock.Acquire(path, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = held.Release() })

	// act
	_, err = filelock.Acquire(path, 50*time.Millisecond)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "another process holds "+path)
}

func TestAcquireGetsTheLockAfterItIsReleased(t *testing.T) {
	t.Parallel()

	// arrange
	path := filepath.Join(t.TempDir(), "lock")

	held, err := filelock.Acquire(path, time.Second)
	require.NoError(t, err)
	require.NoError(t, held.Release())

	// act
	again, err := filelock.Acquire(path, time.Second)

	// assert
	require.NoError(t, err)
	require.NoError(t, again.Release())
}
