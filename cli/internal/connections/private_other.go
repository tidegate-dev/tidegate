//go:build !unix

package connections

// Windows keeps a user's files private through the profile directory's access control list, not through a mode.
func checkOnlyOwnerHasAccess(string, bool) error {
	return nil
}
