//go:build !windows

package supervision

import (
	"errors"
	"os/exec"
	"sync"
	"syscall"
)

func startOwnedProcess(command *exec.Cmd) (*ownedProcess, error) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return nil, err
	}
	owner := &unixProcessGroup{processID: command.Process.Pid}
	return &ownedProcess{command: command, platform: owner}, nil
}

type unixProcessGroup struct {
	mu        sync.Mutex
	processID int
	closed    bool
}

func (process *unixProcessGroup) RequestStop() error {
	return process.signal(syscall.SIGTERM)
}

func (process *unixProcessGroup) Kill() error {
	return process.signal(syscall.SIGKILL)
}

func (process *unixProcessGroup) Close() error {
	process.mu.Lock()
	defer process.mu.Unlock()
	process.closed = true
	return nil
}

func (process *unixProcessGroup) signal(signal syscall.Signal) error {
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.closed {
		return nil
	}
	err := syscall.Kill(-process.processID, signal)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
