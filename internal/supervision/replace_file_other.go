//go:build !windows

package supervision

import (
	"fmt"
	"os"
	"path/filepath"
)

func replaceFile(from string, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}

	directory, err := os.Open(filepath.Dir(to))
	if err != nil {
		return fmt.Errorf("open registry directory for flush: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("flush registry directory: %w", err)
	}
	return nil
}
