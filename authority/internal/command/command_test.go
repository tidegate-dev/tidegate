package command

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
)

func TestASchemaThatIsTooOldSaysToRunMigrate(t *testing.T) {
	t.Parallel()

	// arrange
	tooOld := fmt.Errorf("%w: it's at version 0", db.ErrSchemaTooOld)

	// act
	err := withMigrateHint(tooOld)

	// assert
	require.ErrorIs(t, err, db.ErrSchemaTooOld)
	assert.ErrorContains(t, err, "Run authority migrate first")
}
