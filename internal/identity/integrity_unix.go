//go:build linux || darwin

package identity

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func checkExecutableIntegrity(path string) error {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("cannot inspect original gh integrity")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || (stat.Uid != 0 && stat.Uid != uint32(os.Getuid())) || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("original gh and its parents must be owned by the operator or root without redirection")
		}
		if info.Mode().Perm()&0022 != 0 && !(info.IsDir() && info.Mode()&os.ModeSticky != 0) {
			return fmt.Errorf("original gh or a parent directory is writable outside the operator")
		}
		if err := checkExtendedPermissions(current); err != nil {
			return err
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}
