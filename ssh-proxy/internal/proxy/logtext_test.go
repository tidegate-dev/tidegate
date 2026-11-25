package proxy

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogTextKeepsAShortErrorAsItIs(t *testing.T) {
	t.Parallel()

	// act
	text := logText(errors.New("no common algorithm"))

	// assert
	assert.Equal(t, "no common algorithm", text)
}

func TestLogTextCutsAnErrorThatIsLongerThanTheLimit(t *testing.T) {
	t.Parallel()

	// arrange
	err := errors.New(strings.Repeat("a", maxLoggedErrorBytes+1))

	// act
	text := logText(err)

	// assert
	assert.Equal(t, strings.Repeat("a", maxLoggedErrorBytes)+"...", text)
}
