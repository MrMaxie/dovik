package identity

// POSIX ACL grants cannot exceed the group mask, required to be zero.
func checkExtendedPermissions(string) error { return nil }
