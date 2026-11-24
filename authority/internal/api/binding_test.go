package api

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestCreateBindingRequestAcceptsTheNamesOfAProjectARoleAndAUser(t *testing.T) {
	t.Parallel()

	// act
	err := validationError(t, &authorityv1.CreateBindingRequest{ProjectName: "team-1", RoleName: "operator-2", UserName: "Alice Example"})

	// assert
	assert.NoError(t, err)
}

func TestCreateBindingRequestRefusesAProjectNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, projectName := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			err := validationError(t, &authorityv1.CreateBindingRequest{ProjectName: projectName, RoleName: "operator", UserName: "alice"})

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "project_name: must start with a lowercase letter")
		})
	}
}

func TestCreateBindingRequestRefusesARoleNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, roleName := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			err := validationError(t, &authorityv1.CreateBindingRequest{ProjectName: "payments", RoleName: roleName, UserName: "alice"})

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "role_name: must start with a lowercase letter")
		})
	}
}

func TestCreateBindingRequestRefusesAnEmptyUserName(t *testing.T) {
	t.Parallel()

	// act
	err := validationError(t, &authorityv1.CreateBindingRequest{ProjectName: "payments", RoleName: "operator"})

	// assert
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorContains(t, err, "user")
}

func TestListBindingsRequestRefusesAProjectNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	// act
	err := validationError(t, &authorityv1.ListBindingsRequest{ProjectName: "Payments"})

	// assert
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorContains(t, err, "project_name: must start with a lowercase letter")
}

func TestCreateBindingResponseHoldsTheBindingWithTheNamesOfItsRoleAndUser(t *testing.T) {
	t.Parallel()

	// arrange
	binding := &model.BindingWithNames{
		ID:        model.BindingID{1},
		ProjectID: model.ProjectID{2},
		RoleName:  "operator",
		UserName:  "alice",
	}

	// act
	response := createBindingResponse(binding)

	// assert
	assert.Equal(t, binding.ID.String(), response.GetId())
	assert.Equal(t, binding.ProjectID.String(), response.GetProjectId())
	assert.Equal(t, "operator", response.GetRoleName())
	assert.Equal(t, "alice", response.GetUserName())
}

func TestListedBindingHoldsTheBindingWithTheNamesOfItsRoleAndUser(t *testing.T) {
	t.Parallel()

	// arrange
	binding := &model.BindingWithNames{
		ID:        model.BindingID{1},
		ProjectID: model.ProjectID{2},
		RoleName:  "operator",
		UserName:  "alice",
	}

	// act
	message := listedBinding(binding)

	// assert
	assert.Equal(t, binding.ID.String(), message.GetId())
	assert.Equal(t, binding.ProjectID.String(), message.GetProjectId())
	assert.Equal(t, "operator", message.GetRoleName())
	assert.Equal(t, "alice", message.GetUserName())
}

func TestBindingErrorAnswersNotFoundForAUserThatDoesNotExist(t *testing.T) {
	t.Parallel()

	// act
	err := bindingError(fmt.Errorf("reading: %w", model.ErrUserNotFound))

	// assert
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.EqualError(t, err, "not_found: user not found")
}

func TestBindingErrorGivesARoleThatDoesNotExistTheCodeOfClientError(t *testing.T) {
	t.Parallel()

	// act
	err := bindingError(model.ErrRoleNotFound)

	// assert
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.EqualError(t, err, "not_found: role not found")
}
