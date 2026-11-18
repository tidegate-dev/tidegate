package secret

import (
	"fmt"
	"log/slog"
)

// Bytes holds a binary value that prints as ****. Only Reveal returns it.
type Bytes struct {
	// A pointer, because fmt prints an unexported field of another struct
	// without calling its methods, and prints a pointer there as an address.
	value *[]byte
}

// NewBytes returns value as a Bytes. It doesn't copy value.
func NewBytes(value []byte) Bytes {
	return Bytes{value: &value}
}

// Reveal returns the value, or nil for the zero Bytes.
func (b Bytes) Reveal() []byte {
	if b.value == nil {
		return nil
	}

	return *b.value
}

// String returns ****.
func (Bytes) String() string {
	return hidden
}

// Format prints **** for every verb, including %#v.
func (Bytes) Format(f fmt.State, _ rune) {
	_, _ = f.Write([]byte(hidden))
}

// LogValue logs ****.
func (Bytes) LogValue() slog.Value {
	return slog.StringValue(hidden)
}

// MarshalText encodes ****, also in JSON.
func (Bytes) MarshalText() ([]byte, error) {
	return []byte(hidden), nil
}
