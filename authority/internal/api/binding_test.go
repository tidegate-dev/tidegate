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
	err := validationError(t, &authorityv1.CreateBindingRequest{Project: "team-1", Role: "operator-2", User: "Alice Example"})

	// assert
	assert.NoError(t, err)
}

func TestCreateBindingRequestRefusesAProjectNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, project := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			err := validationError(t, &authorityv1.CreateBindingRequest{Project: project, Role: "operator", User: "alice"})

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "project: must start with a lowercase letter")
		})
	}
}

func TestCreateBindingRequestRefusesARoleNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	for description, role := range refusedNames() {
		t.Run(description, func(t *testing.T) {
			t.Parallel()

			// act
			err := validationError(t, &authorityv1.CreateBindingRequest{Project: "payments", Role: role, User: "alice"})

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "role: must start with a lowercase letter")
		})
	}
}

func TestCreateBindingRequestRefusesAnEmptyUserName(t *testing.T) {
	t.Parallel()

	// act
	err := validationError(t, &authorityv1.CreateBindingRequest{Project: "payments", Role: "operator"})

	// assert
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorContains(t, err, "user")
}

func TestListBindingsRequestRefusesAProjectNameOfAnotherForm(t *testing.T) {
	t.Parallel()

	// act
	err := validationError(t, &authorityv1.ListBindingsRequest{Project: "Payments"})

	// assert
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorContains(t, err, "project: must start with a lowercase letter")
}

func TestBindingMessageHoldsTheBindingWithTheNamesOfItsRoleAndUser(t *testing.T) {
	t.Parallel()

	// arrange
	binding := &model.BindingWithNames{
		ID:        model.BindingID{1},
		ProjectID: model.ProjectID{2},
		RoleName:  "operator",
		UserName:  "alice",
	}

	// act
	message := bindingMessage(binding)

	// assert
	assert.Equal(t, binding.ID.String(), message.GetId())
	assert.Equal(t, binding.ProjectID.String(), message.GetProjectId())
	assert.Equal(t, "operator", message.GetRole())
	assert.Equal(t, "alice", message.GetUser())
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
