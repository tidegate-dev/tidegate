package command

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

func tableText(header []string, rows [][]string) string {
	var text strings.Builder

	table := tabwriter.NewWriter(&text, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(table, strings.Join(header, "\t"))

	for _, row := range rows {
		_, _ = fmt.Fprintln(table, strings.Join(row, "\t"))
	}

	_ = table.Flush()

	return strings.TrimSuffix(text.String(), "\n")
}
