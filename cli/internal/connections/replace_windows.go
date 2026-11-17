package connections

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"
)

// The syscall package has no name for Windows error 32, ERROR_SHARING_VIOLATION.
const errSharingViolation = syscall.Errno(32)

const (
	replaceWaitLimit = 2 * time.Second
	replacePause     = 10 * time.Millisecond
)

// replaceFile renames temporaryPath to path. If another process has path
// open, it tries again for up to 2 seconds.
func replaceFile(temporaryPath, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), replaceWaitLimit)
	defer cancel()

	for {
		err := os.Rename(temporaryPath, path)
		// Windows refuses to rename onto a file that another process has open, such as a virus scanner.
		if !errors.Is(err, errSharingViolation) && !errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return err
		}

		select {
		case <-ctx.Done():
			return err
		case <-time.After(replacePause):
		}
	}
}
