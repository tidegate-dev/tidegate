package command

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// tableStyle prints a table without borders or separators. Two spaces
// separate the columns and the header keeps the case it is given in.
var tableStyle = table.Style{
	Name: "tidegate",
	Box: table.BoxStyle{
		PaddingRight: "  ",
	},
	Format: table.FormatOptions{
		Header: text.FormatDefault,
		Row:    text.FormatDefault,
	},
}

func tableText(header []string, rows [][]string) string {
	writer := table.NewWriter()
	writer.SetStyle(tableStyle)
	writer.SuppressTrailingSpaces()
	writer.AppendHeader(tableRow(header))

	for _, row := range rows {
		writer.AppendRow(tableRow(row))
	}

	return writer.Render()
}

func tableRow(cells []string) table.Row {
	row := make(table.Row, 0, len(cells))
	for _, cell := range cells {
		row = append(row, cell)
	}

	return row
}
