//go:build unix

package connections

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// checkOnlyOwnerHasAccess returns an error unless path belongs to the user
// and nobody else can use it. A path that doesn't exist passes.
func checkOnlyOwnerHasAccess(path string, directory bool) error {
	kind, mode := "file", "600"
	if directory {
		kind, mode = "directory", "700"
	}

	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("checking who can use %s: %w", path, err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)

	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("%s is a symbolic link. tidegate keeps session tokens there and only uses a real %s", path, kind)
	case info.IsDir() != directory, !directory && !info.Mode().IsRegular():
		return fmt.Errorf("%s isn't a %s", path, kind)
	case ok && int(stat.Uid) != os.Geteuid():
		return fmt.Errorf("%s belongs to another user. tidegate keeps session tokens there and only uses a %s of your own", path, kind)
	case info.Mode().Perm()&0o077 != 0:
		return fmt.Errorf("other users can use %s, where tidegate keeps session tokens. Run `chmod %s %s`", path, mode, path)
	default:
		return nil
	}
}
