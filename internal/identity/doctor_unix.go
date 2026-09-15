//go:build linux || darwin

package identity

import (
	"fmt"
	"os"
	"syscall"
)

func checkPrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || stat.Uid != uint32(os.Getuid()) || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("directory must be owned by the operator with mode 0700")
	}
	return checkExtendedPermissions(path)
}
