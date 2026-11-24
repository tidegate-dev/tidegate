package secretengine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/secretengine"
)

type fakeEngine struct {
	label string
}

func (*fakeEngine) Store(context.Context, secret.Bytes) (secretengine.Reference, error) {
	return "", nil
}

func (*fakeEngine) Fetch(context.Context, secretengine.Reference) (secret.Bytes, error) {
	return secret.Bytes{}, nil
}

func (*fakeEngine) Delete(context.Context, secretengine.Reference) error {
	return nil
}

func TestByNameReturnsTheEngineWithThatName(t *testing.T) {
	t.Parallel()

	// arrange
	first := &fakeEngine{label: "first"}
	second := &fakeEngine{label: "second"}
	engines := secretengine.NewEngines(map[string]secretengine.Engine{
		"first":  first,
		"second": second,
	})

	// act
	engine, err := engines.ByName("second")

	// assert
	require.NoError(t, err)
	assert.Same(t, second, engine)
}

func TestByNameReturnsErrUnknownEngineForANameWithoutAnEngine(t *testing.T) {
	t.Parallel()

	// arrange
	engines := secretengine.NewEngines(map[string]secretengine.Engine{
		"first": &fakeEngine{label: "first"},
	})

	// act
	_, err := engines.ByName("second")

	// assert
	require.ErrorIs(t, err, secretengine.ErrUnknownEngine)
}
