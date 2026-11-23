// Package output writes a command's result as text, JSON or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf16"

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

// Print writes result to w in format. It escapes every character that a
// terminal would interpret, apart from line breaks and tabs. JSON and YAML
// still read back as the same text.
func Print(w io.Writer, format Format, result Result) error {
	var (
		text string
		err  error
	)

	switch format {
	case Text:
		text = escape(result.String()+"\n", goEscape)
	case JSON:
		text, err = jsonText(result)
	case YAML:
		text, err = yamlText(result)
	default:
		_, err = ParseFormat(string(format))
	}

	if err != nil {
		return err
	}

	_, err = io.WriteString(w, text)

	return err
}

func jsonText(result Result) (string, error) {
	var encoded strings.Builder

	encoder := json.NewEncoder(&encoded)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(result)
	if err != nil {
		return "", err
	}

	// encoding/json writes a character that isn't graphic only inside a string, where JSON allows \u escapes.
	return escape(encoded.String(), jsonEscape), nil
}

func yamlText(result Result) (string, error) {
	var wrapped yaml.Node

	// Node.Encode parses what yaml.v3 writes, and that fails for some block scalars, like "\t\n". Flow style has none.
	err := wrapped.Encode(struct {
		Result Result `yaml:",flow"`
	}{result})
	if err != nil {
		return "", err
	}

	document := wrapped.Content[1]
	blockStyle(document)

	var encoded strings.Builder

	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)

	err = encoder.Encode(document)
	if err != nil {
		return "", err
	}

	err = encoder.Close()
	if err != nil {
		return "", err
	}

	return escape(encoded.String(), goEscape), nil
}

// blockStyle takes node and everything under it out of flow style. The double
// quotes of the encoder stay, because a YAML 1.1 parser reads a plain yes or
// on as a boolean. A text with a character that isn't graphic gets double
// quotes, because YAML allows escapes only there.
func blockStyle(node *yaml.Node) {
	node.Style &= yaml.DoubleQuotedStyle

	if node.Kind == yaml.ScalarNode && strings.ContainsFunc(node.Value, isNotGraphic) {
		node.Style = yaml.DoubleQuotedStyle
	}

	for _, child := range node.Content {
		blockStyle(child)
	}
}

// A terminal draws graphic characters, which are letters, marks, numbers, punctuation, symbols and spaces.
func isNotGraphic(char rune) bool {
	return !strconv.IsGraphic(char)
}

// escape replaces every character in text that a terminal would interpret,
// apart from line breaks and tabs, by what notation returns for it.
func escape(text string, notation func(rune) string) string {
	var escaped strings.Builder

	for _, char := range text {
		if char == '\n' || char == '\t' || !isNotGraphic(char) {
			escaped.WriteRune(char)
		} else {
			escaped.WriteString(notation(char))
		}
	}

	return escaped.String()
}

// goEscape returns char as Go writes it in a string, like \x1b or \u202e.
// YAML reads these escapes the same way.
func goEscape(char rune) string {
	return strings.Trim(strconv.QuoteRune(char), "'")
}

// jsonEscape returns char as a \u escape. JSON wants a character above U+FFFF
// as the two halves of its UTF-16 form.
func jsonEscape(char rune) string {
	high, low := utf16.EncodeRune(char)
	if utf16.IsSurrogate(high) {
		return fmt.Sprintf(`\u%04x\u%04x`, high, low)
	}

	return fmt.Sprintf(`\u%04x`, char)
}
