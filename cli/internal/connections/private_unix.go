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
func checkOnlyOwnerHasAccess(path string, wantDirectory bool) error {
	kind, chmodMode := "file", "600"
	if wantDirectory {
		kind, chmodMode = "directory", "700"
	}

	// Lstat reports a symbolic link itself, where Stat reports what it points to.
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("checking who can use %s: %w", path, err)
	}

	owner, ownerKnown := info.Sys().(*syscall.Stat_t)
	isSymbolicLink := info.Mode()&fs.ModeSymlink != 0
	isWantedKind := info.IsDir() == wantDirectory && (wantDirectory || info.Mode().IsRegular())
	othersHaveAccess := info.Mode().Perm()&0o077 != 0

	switch {
	case isSymbolicLink:
		return fmt.Errorf("%s is a symbolic link. tidegate keeps session tokens there and only uses a real %s", path, kind)
	case !isWantedKind:
		return fmt.Errorf("%s isn't a %s", path, kind)
	case ownerKnown && int(owner.Uid) != os.Geteuid():
		return fmt.Errorf("%s belongs to another user. tidegate keeps session tokens there and only uses a %s of your own", path, kind)
	case othersHaveAccess:
		return fmt.Errorf("other users can use %s, where tidegate keeps session tokens. Run `chmod %s %s`", path, chmodMode, path)
	default:
		return nil
	}
}
