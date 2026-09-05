//go:build windows

package daemonlaunch

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func startDetached(executable string, arguments, environment []string) (int, error) {
	pid, err := startDetachedWithFlags(executable, arguments, environment, windows.DETACHED_PROCESS|windows.CREATE_NEW_PROCESS_GROUP|windows.CREATE_BREAKAWAY_FROM_JOB)
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return startDetachedWithFlags(executable, arguments, environment, windows.DETACHED_PROCESS|windows.CREATE_NEW_PROCESS_GROUP)
	}
	return pid, err
}

func startDetachedWithFlags(executable string, arguments, environment []string, flags uint32) (int, error) {
	command := exec.Command(executable, arguments...)
	if environment != nil {
		command.Env = environment
	}
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags:    flags,
		HideWindow:       true,
		NoInheritHandles: true,
	}
	if err := command.Start(); err != nil {
		return 0, fmt.Errorf("start detached daemon: %w", err)
	}
	pid := command.Process.Pid
	if err := command.Process.Release(); err != nil {
		_ = command.Process.Kill()
		return 0, fmt.Errorf("release detached daemon: %w", err)
	}
	return pid, nil
}
