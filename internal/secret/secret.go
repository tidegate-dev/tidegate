// Package secret keeps values like passwords and tokens out of logs, error
// messages and JSON. It doesn't wipe them from memory, which Go can't do
// reliably for strings.
package secret

import (
	"fmt"
	"log/slog"
)

const hidden = "****"

// String holds a value that prints as ****. Only Reveal returns it.
type String struct {
	// A pointer, because fmt prints an unexported field of another struct
	// without calling its methods, and prints a pointer there as an address.
	value *string
}

// New returns value as a String.
func New(value string) String {
	return String{value: &value}
}

// Reveal returns the value, or "" for the zero String.
func (s String) Reveal() string {
	if s.value == nil {
		return ""
	}

	return *s.value
}

// String returns ****.
func (String) String() string {
	return hidden
}

// Format prints **** for every verb, including %#v.
func (String) Format(f fmt.State, _ rune) {
	_, _ = f.Write([]byte(hidden))
}

// LogValue logs ****.
func (String) LogValue() slog.Value {
	return slog.StringValue(hidden)
}

// MarshalText encodes ****, also in JSON.
func (String) MarshalText() ([]byte, error) {
	return []byte(hidden), nil
}
