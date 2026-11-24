package output

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// labeled is a result with a text that the server chose, as a value and as a
// key. Its String returns the text without quotes, which a command's result
// must not do.
type labeled struct {
	Name   string            `json:"name"   yaml:"name"`
	Labels map[string]string `json:"labels" yaml:"labels"`
}

func (result labeled) String() string { return result.Name }

type listed struct {
	Names  []string          `json:"names"  yaml:"names"`
	Labels map[string]string `json:"labels" yaml:"labels"`
}

func (result listed) String() string { return "" }

func newLabeled(text string) labeled {
	return labeled{Name: text, Labels: map[string]string{text: text}}
}

func addTexts(f *testing.F) {
	f.Helper()

	for _, text := range []string{
		"a\x1b[2Jb",
		"a\u009b2Jb",
		"a\u0085b",
		"a\x7fb",
		"a\tb",
		"a\rb",
		"a\u202eb",
		"a\u200bb",
		"a\u2028b",
		"a\ufeffb",
		"a\u00a0b",
		"a\U000e0001b",
		"a\n\x1b[2Jb",
		"\t\n",
		"<<",
		"true",
		" a: b # c",
		"",
	} {
		f.Add(text)
	}
}

func FuzzPrintWritesNoCharacterThatATerminalInterprets(f *testing.F) {
	addTexts(f)

	f.Fuzz(func(t *testing.T, text string) {
		for _, format := range []Format{Text, JSON, YAML} {
			var written bytes.Buffer

			err := Print(&written, format, newLabeled(text))

			require.NoError(t, err)
			assert.Truef(t, utf8.Valid(written.Bytes()), "%s wrote invalid UTF-8 for %q", format, text)

			for _, char := range written.String() {
				assert.Truef(t, char == '\n' || char == '\t' || strconv.IsGraphic(char), "%s wrote %U for %q", format, char, text)
			}
		}
	})
}

func FuzzPrintWritesJSONAndYAMLThatReadBackAsTheSameText(f *testing.F) {
	addTexts(f)

	f.Fuzz(func(t *testing.T, text string) {
		// A protobuf string, which is where these texts come from, is always valid UTF-8.
		if !utf8.ValidString(text) {
			t.Skip("not valid UTF-8")
		}

		for format, decode := range map[Format]func([]byte, any) error{JSON: json.Unmarshal, YAML: yaml.Unmarshal} {
			var written bytes.Buffer

			err := Print(&written, format, newLabeled(text))

			require.NoError(t, err)

			var read labeled

			require.NoErrorf(t, decode(written.Bytes(), &read), "%s wrote %q", format, written.String())
			assert.Equalf(t, newLabeled(text), read, "%s wrote %q", format, written.String())
		}
	})
}

func TestPrintKeepsLettersAndSpacesOutsideASCIIAsTheyAre(t *testing.T) {
	t.Parallel()

	for _, format := range []Format{Text, JSON, YAML} {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			// arrange
			var written bytes.Buffer

			// act
			err := Print(&written, format, newLabeled("Zoë \u3000日本"))

			// assert
			require.NoError(t, err)
			assert.Contains(t, written.String(), "Zoë \u3000日本")
		})
	}
}

func TestPrintWritesYAMLInBlockStyle(t *testing.T) {
	t.Parallel()

	// arrange
	var written bytes.Buffer

	result := listed{Names: []string{"web-1", "web-2"}, Labels: map[string]string{"env": "prod"}}

	// act
	err := Print(&written, YAML, result)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "names:\n  - web-1\n  - web-2\nlabels:\n  env: prod\n", written.String())
}

func TestPrintQuotesAYAMLTextThatAYAML11ParserReadsAsABoolean(t *testing.T) {
	t.Parallel()

	// arrange
	var written bytes.Buffer

	// act
	err := Print(&written, YAML, newLabeled("yes"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, "name: \"yes\"\nlabels:\n  \"yes\": \"yes\"\n", written.String())
}
