package supervision

import (
	"errors"
	"fmt"
	"os/exec"
)

type ownedProcess struct {
	command  *exec.Cmd
	platform platformProcess
}

type platformProcess interface {
	RequestStop() error
	Kill() error
	Close() error
}

type processExit struct {
	Code int
	Err  error
}

func (process *ownedProcess) PID() int {
	return process.command.Process.Pid
}

func (process *ownedProcess) Wait() processExit {
	err := process.command.Wait()
	exit := processExit{Code: -1, Err: err}
	if process.command.ProcessState != nil {
		exit.Code = process.command.ProcessState.ExitCode()
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		exit.Err = nil
	}
	if closeErr := process.platform.Close(); closeErr != nil && exit.Err == nil {
		exit.Err = fmt.Errorf("close process owner: %w", closeErr)
	}
	return exit
}

func newCommand(definition ProcessDefinition) *exec.Cmd {
	command := exec.Command(definition.Command, definition.Arguments...)
	command.Dir = definition.WorkingDirectory
	if command.Dir == "" {
		command.Dir = "."
	}
	command.Env = appendEnvironment(definition.EnvironmentOverrides)
	return command
}
