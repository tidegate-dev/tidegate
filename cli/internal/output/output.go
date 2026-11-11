// Package output writes a command's result as text, JSON or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// Format is how Print writes a result.
type Format string

// The formats that Print supports.
const (
	Text Format = "text"
	JSON Format = "json"
	YAML Format = "yaml"
)

// Result is what a command prints when it succeeds. Its String method gives
// the text format.
type Result interface {
	fmt.Stringer
}

// ParseFormat returns the format named value, or an error if there is none.
func ParseFormat(value string) (Format, error) {
	switch format := Format(value); format {
	case Text, JSON, YAML:
		return format, nil
	default:
		return "", fmt.Errorf("unknown output format %q, want text, json or yaml", value)
	}
}

// Print writes result to w in format.
func Print(w io.Writer, format Format, result Result) error {
	switch format {
	case Text:
		_, err := fmt.Fprintln(w, result.String())

		return err
	case JSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")

		return encoder.Encode(result)
	case YAML:
		encoder := yaml.NewEncoder(w)
		encoder.SetIndent(2)

		err := encoder.Encode(result)
		if err != nil {
			return err
		}

		return encoder.Close()
	default:
		return fmt.Errorf("unknown output format %q, want text, json or yaml", format)
	}
}
