//go:build windows

package supervision

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func startOwnedProcess(command *exec.Cmd) (*ownedProcess, error) {
	job, err := createKillOnCloseJob()
	if err != nil {
		return nil, err
	}
	owner := &windowsProcess{job: job}
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_SUSPENDED | windows.CREATE_NO_WINDOW}
	if err := command.Start(); err != nil {
		owner.Close()
		return nil, err
	}

	processHandle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		uint32(command.Process.Pid),
	)
	if err != nil {
		cleanupSuspendedProcess(command, owner)
		return nil, fmt.Errorf("open process for job assignment: %w", err)
	}
	if err := windows.AssignProcessToJobObject(job, processHandle); err != nil {
		windows.CloseHandle(processHandle)
		cleanupSuspendedProcess(command, owner)
		return nil, fmt.Errorf("assign process to job: %w", err)
	}
	windows.CloseHandle(processHandle)
	if err := resumeProcessThreads(uint32(command.Process.Pid)); err != nil {
		cleanupSuspendedProcess(command, owner)
		return nil, err
	}

	return &ownedProcess{command: command, platform: owner}, nil
}

type windowsProcess struct {
	mu     sync.Mutex
	job    windows.Handle
	closed bool
}

func (process *windowsProcess) RequestStop() error {
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.closed {
		return nil
	}
	return windows.TerminateJobObject(process.job, 1)
}

func (process *windowsProcess) Kill() error {
	return process.RequestStop()
}

func (process *windowsProcess) Close() error {
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.closed {
		return nil
	}
	process.closed = true
	return windows.CloseHandle(process.job)
}

func createKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("create job object: %w", err)
	}
	information := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	information.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&information)),
		uint32(unsafe.Sizeof(information)),
	)
	if err != nil {
		windows.CloseHandle(job)
		return 0, fmt.Errorf("configure job object: %w", err)
	}
	return job, nil
}

func resumeProcessThreads(processID uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fmt.Errorf("snapshot process threads: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	err = windows.Thread32First(snapshot, &entry)
	resumed := 0
	for err == nil {
		if entry.OwnerProcessID == processID {
			thread, openErr := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
			if openErr != nil {
				return fmt.Errorf("open suspended process thread: %w", openErr)
			}
			_, resumeErr := windows.ResumeThread(thread)
			windows.CloseHandle(thread)
			if resumeErr != nil {
				return fmt.Errorf("resume process thread: %w", resumeErr)
			}
			resumed++
		}
		err = windows.Thread32Next(snapshot, &entry)
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return fmt.Errorf("enumerate process threads: %w", err)
	}
	if resumed == 0 {
		return fmt.Errorf("no suspended process thread found")
	}
	return nil
}

func cleanupSuspendedProcess(command *exec.Cmd, owner *windowsProcess) {
	_ = command.Process.Kill()
	_ = command.Wait()
	_ = owner.Close()
}
