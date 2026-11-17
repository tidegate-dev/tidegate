//go:build !windows

package connections

import "os"

// replaceFile renames temporaryPath to path.
func replaceFile(temporaryPath, path string) error {
	return os.Rename(temporaryPath, path)
}
