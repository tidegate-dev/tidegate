package secret_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

const value = "correct horse battery staple"

func TestRevealReturnsTheValue(t *testing.T) {
	t.Parallel()

	// act
	revealed := secret.New(value).Reveal()

	// assert
	assert.Equal(t, value, revealed)
}

func TestRevealReturnsAnEmptyStringForTheZeroString(t *testing.T) {
	t.Parallel()

	// act
	revealed := secret.String{}.Reveal()

	// assert
	assert.Empty(t, revealed)
}

func TestFormattingHidesTheValue(t *testing.T) {
	t.Parallel()

	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d"} {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()

			// act
			formatted := fmt.Sprintf(verb, secret.New(value))

			// assert
			assert.Equal(t, "****", formatted)
		})
	}
}

func TestFormattingAStructHidesTheValue(t *testing.T) {
	t.Parallel()

	// arrange
	login := struct {
		Name     string
		Password secret.String
	}{Name: "alice", Password: secret.New(value)}

	// act
	formatted := fmt.Sprintf("%+v %#v", login, login)

	// assert
	assert.NotContains(t, formatted, value)
}

func TestFormattingAStructHidesTheValueInAnUnexportedField(t *testing.T) {
	t.Parallel()

	// arrange
	// fmt doesn't call Format or String on an unexported field.
	login := struct {
		name     string
		password secret.String
	}{name: "alice", password: secret.New(value)}

	var logs bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logs, nil))

	// act
	formatted := fmt.Sprintf("%v %+v %#v", login, login, login)

	logger.Info("login", "login", login)

	// assert
	assert.NotContains(t, formatted, value)
	assert.NotContains(t, logs.String(), value)
}

func TestLoggingHidesTheValue(t *testing.T) {
	t.Parallel()

	for name, newHandler := range map[string]func(*bytes.Buffer) slog.Handler{
		"text": func(w *bytes.Buffer) slog.Handler { return slog.NewTextHandler(w, nil) },
		"json": func(w *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(w, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			var logs bytes.Buffer

			logger := slog.New(newHandler(&logs))

			// act
			logger.Info("login", "password", secret.New(value))

			// assert
			assert.NotContains(t, logs.String(), value)
			assert.Contains(t, logs.String(), "****")
		})
	}
}

func TestJSONHidesTheValue(t *testing.T) {
	t.Parallel()

	// act
	encoded, err := json.Marshal(map[string]secret.String{"password": secret.New(value)})

	// assert
	require.NoError(t, err)
	assert.JSONEq(t, `{"password":"****"}`, string(encoded))
}
