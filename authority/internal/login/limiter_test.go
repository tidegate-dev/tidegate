package login

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiterLetsTheLimitThrough(t *testing.T) {
	t.Parallel()

	// arrange
	limit := newLimiter(2)

	// act
	first := limit.acquire(t.Context())
	second := limit.acquire(t.Context())

	// assert
	require.NoError(t, first)
	require.NoError(t, second)
}

func TestLimiterWaitsUntilTheContextEndsWhenFull(t *testing.T) {
	t.Parallel()

	// arrange
	limit := newLimiter(1)
	require.NoError(t, limit.acquire(t.Context()))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// act
	err := limit.acquire(ctx)

	// assert
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLimiterLetsTheNextOneThroughAfterARelease(t *testing.T) {
	t.Parallel()

	// arrange
	limit := newLimiter(1)
	require.NoError(t, limit.acquire(t.Context()))

	// act
	limit.release()
	err := limit.acquire(t.Context())

	// assert
	require.NoError(t, err)
}
