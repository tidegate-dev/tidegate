package certreload

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"
)

// ErrKeyFileAccess means that users other than root and the one tidegate runs
// as could read or change the certificate's key file.
var ErrKeyFileAccess = errors.New("the TLS certificate's key file has an unsafe owner or mode")

const rootUID = 0

// readKeyFile checks the open file instead of the path, so nobody can swap the
// file between the check and the read.
func readKeyFile(keyFile string) ([]byte, error) {
	file, err := os.Open(keyFile) //nolint:gosec // The operator sets the path.
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("%s: this system doesn't say who owns the file", keyFile)
	}

	err = checkKeyFileAccess(int(stat.Uid), os.Geteuid(), info.Mode().Perm())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", keyFile, err)
	}

	return io.ReadAll(file)
}

// Kubernetes makes root the owner of a Secret's files and gives them the
// Pod's fsGroup as their group. That's why checkKeyFileAccess lets root's
// group read the key.
func checkKeyFileAccess(owner, self int, mode fs.FileMode) error {
	switch owner {
	case self:
		if mode&0o077 != 0 {
			return fmt.Errorf("%w: it has mode %04o, and a key file owned by the user tidegate runs as (%d) needs mode 0600 or 0400", ErrKeyFileAccess, mode, self)
		}
	case rootUID:
		if mode&0o037 != 0 {
			return fmt.Errorf("%w: it has mode %04o, and a key file owned by root needs mode 0640 or 0440", ErrKeyFileAccess, mode)
		}
	default:
		return fmt.Errorf("%w: user %d owns it, and only root or the user tidegate runs as (%d) may own it", ErrKeyFileAccess, owner, self)
	}

	return nil
}
