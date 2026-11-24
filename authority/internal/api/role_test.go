package api

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestCreateRoleRequestAcceptsNamesOfLowercaseLettersDigitsAndDashes(t *testing.T) {
	t.Parallel()

	// act
	err := validationError(t, &authorityv1.CreateRoleRequest{ProjectName: "team-1", Name: "operator-2"})

	// assert
	assert.NoError(t, err)
}

func TestCreateRoleRequestRefusesARoleNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, name := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			err := validationError(t, &authorityv1.CreateRoleRequest{ProjectName: "payments", Name: name})

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "name: must start with a lowercase letter")
		})
	}
}

func TestCreateRoleRequestRefusesAProjectNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, project := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			err := validationError(t, &authorityv1.CreateRoleRequest{ProjectName: project, Name: "operator"})

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "project_name: must start with a lowercase letter")
		})
	}
}

func TestPermissionsTakesEachPermissionFromItsOwnField(t *testing.T) {
	t.Parallel()

	for description, test := range map[string]struct {
		message *authorityv1.ProjectPermissions
		want    model.ProjectPermissions
	}{
		"no message":      {nil, model.ProjectPermissions{}},
		"create roles":    {&authorityv1.ProjectPermissions{CreateRoles: true}, model.ProjectPermissions{CreateRoles: true}},
		"escalate":        {&authorityv1.ProjectPermissions{Escalate: true}, model.ProjectPermissions{Escalate: true}},
		"create bindings": {&authorityv1.ProjectPermissions{CreateBindings: true}, model.ProjectPermissions{CreateBindings: true}},
		"bind":            {&authorityv1.ProjectPermissions{Bind: true}, model.ProjectPermissions{Bind: true}},
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			got := permissions(test.message)

			// assert
			assert.Equal(t, test.want, got)
		})
	}
}

func TestCreateRoleResponseGivesEachPermissionItsOwnField(t *testing.T) {
	t.Parallel()

	for description, granted := range map[string]model.ProjectPermissions{
		"create roles":    {CreateRoles: true},
		"escalate":        {Escalate: true},
		"create bindings": {CreateBindings: true},
		"bind":            {Bind: true},
	} {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			response := createRoleResponse(&model.Role{Permissions: granted})

			// assert
			assert.Equal(t, granted, permissions(response.GetPermissions()))
		})
	}
}

// refusedNames holds one name for each way of breaking the rule that project,
// role and asset names share.
func refusedNames() map[string]string {
	return map[string]string{
		"empty":              "",
		"uppercase letter":   "Operator",
		"digit at the start": "1operator",
		"dash at the end":    "operator-",
		"underscore":         "operator_1",
		"space inside":       "operator 1",
		"64 characters":      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}
