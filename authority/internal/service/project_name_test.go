package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateProjectNameAcceptsLowercaseLettersDigitsAndDashes(t *testing.T) {
	t.Parallel()

	for name, projectName := range map[string]string{
		"one letter":        "a",
		"letters":           "payments",
		"digit at the end":  "team1",
		"dash inside":       "team-1",
		"two dashes inside": "team--1",
		"63 characters":     strings.Repeat("a", 63),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			err := ValidateProjectName(projectName)

			// assert
			assert.NoError(t, err)
		})
	}
}

func TestValidateProjectNameRefusesEveryOtherName(t *testing.T) {
	t.Parallel()

	for name, projectName := range map[string]string{
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
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			err := ValidateProjectName(projectName)

			// assert
			assert.ErrorIs(t, err, ErrInvalidProjectName)
		})
	}
}
