//go:build windows

package supervision

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func replaceFile(from string, to string) error {
	fromPointer, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return fmt.Errorf("encode source path: %w", err)
	}
	toPointer, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return fmt.Errorf("encode destination path: %w", err)
	}
	flags := uint32(windows.MOVEFILE_REPLACE_EXISTING | windows.MOVEFILE_WRITE_THROUGH)
	if err := windows.MoveFileEx(fromPointer, toPointer, flags); err != nil {
		return fmt.Errorf("move file with replacement: %w", err)
	}
	return nil
}
