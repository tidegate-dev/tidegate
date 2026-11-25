package secret_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemera-dev/phemera/internal/secret"
)

// renderings returns the forms in which fmt and slog print raw.
func renderings(raw []byte) []string {
	decimal := strings.Trim(fmt.Sprintf("%d", raw), "[]")
	hexadecimal := strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%#v", raw), "[]byte{"), "}")

	return []string{string(raw), decimal, hexadecimal, fmt.Sprintf("%x", raw)}
}

func TestBytesRevealReturnsTheValue(t *testing.T) {
	t.Parallel()

	// act
	revealed := secret.NewBytes([]byte(value)).Reveal()

	// assert
	assert.Equal(t, []byte(value), revealed)
}

func TestBytesRevealReturnsNoBytesForTheZeroBytes(t *testing.T) {
	t.Parallel()

	// act
	revealed := secret.Bytes{}.Reveal()

	// assert
	assert.Empty(t, revealed)
}

func TestFormattingBytesHidesTheValue(t *testing.T) {
	t.Parallel()

	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d"} {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()

			// act
			formatted := fmt.Sprintf(verb, secret.NewBytes([]byte(value)))

			// assert
			assert.Equal(t, "****", formatted)
		})
	}
}

func TestFormattingAStructHidesTheBytes(t *testing.T) {
	t.Parallel()

	// arrange
	session := struct {
		Name      string
		TokenHash secret.Bytes
	}{Name: "alice", TokenHash: secret.NewBytes([]byte(value))}

	// act
	formatted := fmt.Sprintf("%+v %#v", session, session)

	// assert
	for _, rendering := range renderings([]byte(value)) {
		assert.NotContains(t, formatted, rendering)
	}
}

func TestFormattingAStructHidesTheBytesInAnUnexportedField(t *testing.T) {
	t.Parallel()

	// arrange
	// fmt doesn't call Format or String on an unexported field.
	session := struct {
		name      string
		tokenHash secret.Bytes
	}{name: "alice", tokenHash: secret.NewBytes([]byte(value))}

	var logs bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logs, nil))

	// act
	formatted := fmt.Sprintf("%v %+v %#v", session, session, session)

	logger.Info("session", "session", session)

	// assert
	for _, rendering := range renderings([]byte(value)) {
		assert.NotContains(t, formatted, rendering)
		assert.NotContains(t, logs.String(), rendering)
	}
}

func TestLoggingHidesTheBytes(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		newHandler func(*bytes.Buffer) slog.Handler
		logged     string
	}{
		"text": {
			newHandler: func(w *bytes.Buffer) slog.Handler { return slog.NewTextHandler(w, nil) },
			logged:     `tokenHash=****`,
		},
		"json": {
			newHandler: func(w *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(w, nil) },
			logged:     `"tokenHash":"****"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			var logs bytes.Buffer

			logger := slog.New(test.newHandler(&logs))

			// act
			logger.Info("session", "tokenHash", secret.NewBytes([]byte(value)))

			// assert
			assert.Contains(t, logs.String(), test.logged)
		})
	}
}

func TestJSONHidesTheBytes(t *testing.T) {
	t.Parallel()

	// act
	encoded, err := json.Marshal(map[string]secret.Bytes{"tokenHash": secret.NewBytes([]byte(value))})

	// assert
	require.NoError(t, err)
	assert.JSONEq(t, `{"tokenHash":"****"}`, string(encoded))
}
