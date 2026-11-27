package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableTextAlignsColumnsWithTwoSpaces(t *testing.T) {
	t.Parallel()

	// act
	text := tableText([]string{"ID", "NAME"}, [][]string{
		{"1", "payments"},
		{"12345", "db"},
	})

	// assert
	assert.Equal(t, "ID     NAME\n"+
		"1      payments\n"+
		"12345  db", text)
}

func TestTableTextKeepsTheCaseOfHeaderAndCells(t *testing.T) {
	t.Parallel()

	// act
	text := tableText([]string{"Id", "name"}, [][]string{{"aBc", "Def"}})

	// assert
	assert.Equal(t, "Id   name\naBc  Def", text)
}

func TestTableTextLeavesNoTrailingSpacesAfterAnEmptyLastCell(t *testing.T) {
	t.Parallel()

	// act
	text := tableText([]string{"NAME", "LABELS"}, [][]string{{"bastion", ""}})

	// assert
	assert.Equal(t, "NAME     LABELS\nbastion", text)
}

func TestTableTextPrintsOnlyTheHeaderWithoutRows(t *testing.T) {
	t.Parallel()

	// act
	text := tableText([]string{"ID", "NAME"}, nil)

	// assert
	assert.Equal(t, "ID  NAME", text)
}
