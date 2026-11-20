package command

import (
	"testing"

	"github.com/stretchr/testify/assert"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestBindingResultTextShowsTheRoleTheUserAndTheID(t *testing.T) {
	t.Parallel()

	// arrange
	result := newBindingResult(&authorityv1.Binding{Id: "1", ProjectId: "2", Role: "operator", User: "alice"})

	// act
	text := result.String()

	// assert
	assert.Equal(t, "operator\talice\t1", text)
}

func TestBindingResultTextQuotesAUserNameThatATerminalWouldInterpret(t *testing.T) {
	t.Parallel()

	// arrange
	result := bindingResult{ID: "1", Role: "operator", User: "a\x1b[2Jb"}

	// act
	text := result.String()

	// assert
	assert.NotContains(t, text, "\x1b")
	assert.Contains(t, text, `"a\x1b[2Jb"`)
}

func TestBindingsResultTextShowsTheRoleTheUserAndTheIDOfEachBinding(t *testing.T) {
	t.Parallel()

	// arrange
	result := newBindingsResult([]*authorityv1.Binding{
		{Id: "1", Role: "auditor", User: "bob"},
		{Id: "2", Role: "operator", User: "alice"},
	})

	// act
	text := result.String()

	// assert
	assert.Equal(t, "ROLE      USER   ID\n"+
		"auditor   bob    1\n"+
		"operator  alice  2", text)
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
	result := newBindingsResult([]*authorityv1.Binding{{Id: "1", Role: "operator", User: "a\x1b[2Jb"}})

	// act
	text := result.String()

	// assert
	assert.NotContains(t, text, "\x1b")
	assert.Contains(t, text, `"a\x1b[2Jb"`)
}
