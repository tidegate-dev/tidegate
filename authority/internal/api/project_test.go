package api

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/stretchr/testify/assert"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func TestCreateProjectRequestAcceptsANameOfLowercaseLettersDigitsAndDashes(t *testing.T) {
	t.Parallel()

	for description, input := range map[string]string{
		"one letter":        "a",
		"letters":           "payments",
		"digit at the end":  "team1",
		"dash inside":       "team-1",
		"two dashes inside": "team--1",
		"63 characters":     strings.Repeat("a", 63),
	} {
		t.Run(fmt.Sprintf("%s %+q", description, input), func(t *testing.T) {
			t.Parallel()

			// act
			err := checkCreateProjectRequest(t, input)

			// assert
			assert.NoError(t, err)
		})
	}
}

func TestCreateProjectRequestRefusesEveryOtherName(t *testing.T) {
	t.Parallel()

	for description, input := range map[string]string{
		"empty":               "",
		"uppercase letter":    "Payments",
		"digit at the start":  "1team",
		"dash at the start":   "-team",
		"dash at the end":     "team-",
		"underscore":          "team_1",
		"space inside":        "team 1",
		"space at the start":  " team",
		"newline at the end":  "team\n",
		"letter with umlaut":  "päyments",
		"Cyrillic lookalike":  "p\u0430yments",
		"zero-width space":    "payments\u200b",
		"NUL":                 "pay\x00ments",
		"escape sequence":     "pay\x1b[2Jments",
		"64 characters":       strings.Repeat("a", 64),
		"64 with a dash":      "a" + strings.Repeat("-", 62) + "a",
		"only a dash":         "-",
		"dot between letters": "team.one",
	} {
		t.Run(fmt.Sprintf("%s %+q", description, input), func(t *testing.T) {
			t.Parallel()

			// act
			err := checkCreateProjectRequest(t, input)

			// assert
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.ErrorContains(t, err, "must start with a lowercase letter")
		})
	}
}

// checkCreateProjectRequest returns what the interceptor that the server
// uses answers to a CreateProjectRequest with the name.
func checkCreateProjectRequest(t *testing.T, name string) error {
	t.Helper()

	handled := func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&authorityv1.CreateProjectResponse{}), nil
	}
	request := connect.NewRequest(&authorityv1.CreateProjectRequest{Name: name})

	_, err := validate.NewInterceptor().WrapUnary(handled)(t.Context(), request)

	return err
}
