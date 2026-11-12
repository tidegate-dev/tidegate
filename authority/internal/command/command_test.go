package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpWorksWithoutConfig(t *testing.T) {
	for _, args := range [][]string{
		{"authority", "--help"},
		{"authority", "migrate", "--help"},
		{"authority", "create-initial-admin", "--help"},
	} {
		t.Run(args[1], func(t *testing.T) {
			// arrange
			t.Setenv("TIDEGATE_AUTHORITY__DATABASE__URL", "")

			var output bytes.Buffer

			command := New()
			command.Writer = &output

			// act
			err := command.Run(t.Context(), args)

			// assert
			require.NoError(t, err)
			assert.Contains(t, output.String(), "USAGE")
		})
	}
}

func TestUnknownArgumentsAreRejected(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"authority", "bogus"},
		{"authority", "migrate", "bogus"},
	} {
		t.Run(args[len(args)-2], func(t *testing.T) {
			t.Parallel()

			// act
			err := New().Run(t.Context(), args)

			// assert
			require.EqualError(t, err, `unknown command "bogus"`)
		})
	}
}

func TestCreateInitialAdminNeedsExactlyOneName(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"no name":   {"authority", "create-initial-admin"},
		"two names": {"authority", "create-initial-admin", "alice", "bob"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			err := New().Run(t.Context(), args)

			// assert
			require.EqualError(t, err, "create-initial-admin takes exactly one argument, the user's name")
		})
	}
}
