package command

import (
	"testing"

	"github.com/stretchr/testify/assert"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestRoleResultTextQuotesANameThatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// arrange
	result := roleResult{ID: "1", Name: "a\x1b[2Jb"}

	// act
	text := result.String()

	// assert
	assert.NotContains(t, text, "\x1b")
	assert.Contains(t, text, `"a\x1b[2Jb"`)
}

func TestNewRoleResultTakesEachPermissionFromItsOwnField(t *testing.T) {
	t.Parallel()

	for description, test := range map[string]struct {
		permissions *authorityv1.ProjectPermissions
		want        permissionsResult
	}{
		"none":            {nil, permissionsResult{}},
		"create roles":    {&authorityv1.ProjectPermissions{CreateRoles: true}, permissionsResult{CreateRoles: true}},
		"escalate":        {&authorityv1.ProjectPermissions{Escalate: true}, permissionsResult{Escalate: true}},
		"create bindings": {&authorityv1.ProjectPermissions{CreateBindings: true}, permissionsResult{CreateBindings: true}},
		"bind":            {&authorityv1.ProjectPermissions{Bind: true}, permissionsResult{Bind: true}},
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			result := newRoleResult(&authorityv1.CreateRoleResponse{Permissions: test.permissions})

			// assert
			assert.Equal(t, test.want, result.Permissions)
		})
	}
}
