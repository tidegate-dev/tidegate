package command

import (
	"testing"

	"github.com/stretchr/testify/assert"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestBindingResultTextShowsTheIDTheRoleAndTheUser(t *testing.T) {
	t.Parallel()

	// arrange
	result := newBindingResult(&authorityv1.CreateBindingResponse{Id: "1", ProjectId: "2", RoleName: "operator", UserName: "alice"})

	// act
	text := result.String()

	// assert
	assert.Equal(t, "1\toperator\talice", text)
}

func TestBindingsResultTextShowsTheIDTheRoleAndTheUserOfEachBinding(t *testing.T) {
	t.Parallel()

	// arrange
	result := newBindingsResult([]*authorityv1.ListBindingsResponse_Binding{
		{Id: "1", RoleName: "auditor", UserName: "bob"},
		{Id: "2", RoleName: "operator", UserName: "alice"},
	})

	// act
	text := result.String()

	// assert
	assert.Equal(t, "ID  ROLE      USER\n"+
		"1   auditor   bob\n"+
		"2   operator  alice", text)
}

func TestBindingsResultTextNamesTheCreateCommandWhenThereAreNoBindings(t *testing.T) {
	t.Parallel()

	// act
	text := newBindingsResult(nil).String()

	// assert
	assert.Contains(t, text, "tidegate binding create")
}

func TestBindingsResultTextQuotesWhatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// arrange
	result := newBindingsResult([]*authorityv1.ListBindingsResponse_Binding{{Id: "1", RoleName: "operator", UserName: "a\x1b[2Jb"}})

	// act
	text := result.String()

	// assert
	assert.NotContains(t, text, "\x1b")
	assert.Contains(t, text, `"a\x1b[2Jb"`)
}
