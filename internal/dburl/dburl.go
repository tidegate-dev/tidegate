// Package dburl parses a Postgres connection URL, with errors that are safe
// to show.
package dburl

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pgx also reads the PG variables, so the message names them too.
var errInvalid = errors.New("isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid")

// pgx's error says which setting names the file only in its text.
var tlsFiles = []struct {
	pgxMessage string
	setting    string
	variable   string
}{
	{pgxMessage: "unable to read CA file", setting: "sslrootcert", variable: "PGSSLROOTCERT"},
	{pgxMessage: "unable to read cert", setting: "sslcert", variable: "PGSSLCERT"},
	{pgxMessage: "unable to read sslkey", setting: "sslkey", variable: "PGSSLKEY"},
}

// Parse parses url like pgxpool.ParseConfig. Its error continues a sentence
// about the URL and holds nothing from it, because pgx can't always find the
// password to mask.
func Parse(url string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, safeError(err)
	}

	return config, nil
}

func safeError(err error) error {
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		return errInvalid
	}

	// pgx wraps the fs.PathError in its description of the file.
	for wrapper := err; wrapper != nil; wrapper = errors.Unwrap(wrapper) {
		for _, file := range tlsFiles {
			if wrapper.Error() == file.pgxMessage+": "+pathErr.Error() {
				return fmt.Errorf("can't be used, because %s or %s names a file that can't be read: %w",
					file.setting, file.variable, pathErr.Err)
			}
		}
	}

	return errInvalid
}
