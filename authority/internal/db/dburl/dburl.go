// Package dburl parses the authority's database URL, with errors that are
// safe to show.
package dburl

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pgx also reads the PG variables, so the message names them too.
var errInvalid = errors.New("isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid")

// tlsFiles maps how pgx describes a TLS file that it can't read to the
// setting that names the file, because pgx's error has no type that says it.
// If pgx changes a description, the error falls back to errInvalid, and the
// tests fail.
var tlsFiles = []struct {
	pgxMessage string
	setting    string
	variable   string
}{
	{pgxMessage: "unable to read CA file", setting: "sslrootcert", variable: "PGSSLROOTCERT"},
	{pgxMessage: "unable to read cert", setting: "sslcert", variable: "PGSSLCERT"},
	{pgxMessage: "unable to read sslkey", setting: "sslkey", variable: "PGSSLKEY"},
}

// Parse parses url like pgxpool.ParseConfig, which also reads the PG
// variables and the TLS files that they or the URL name. Its error leaves out
// pgx's error, because pgx masks the password only where it can find it, and
// a malformed URL can hide it. If a TLS file can't be read, the error names
// the setting and the file, and nothing else from the URL. The error reads
// as the end of a sentence about the URL, as in "the database URL isn't a
// valid ...".
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

	// The error that wraps pathErr is pgx's description of the file. pathErr
	// itself holds only the operation, the path and the system's error.
	for wrapper := err; wrapper != nil; wrapper = errors.Unwrap(wrapper) {
		for _, file := range tlsFiles {
			if wrapper.Error() == file.pgxMessage+": "+pathErr.Error() {
				return fmt.Errorf("can't be used, because %s or %s names a file that can't be read: %w",
					file.setting, file.variable, pathErr)
			}
		}
	}

	return errInvalid
}
