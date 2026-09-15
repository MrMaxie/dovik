//go:build !windows

package identity

import (
	"fmt"
	"os"
	"path/filepath"
)

func protectPath(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("identity storage cannot be a symlink")
	}
	mode := os.FileMode(0600)
	if info.IsDir() {
		mode = 0700
	}
	return os.Chmod(path, mode)
}

func replaceState(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}
	file, err := os.Open(filepath.Dir(to))
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
