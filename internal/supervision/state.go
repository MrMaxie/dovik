package supervision

import (
	"errors"
	"fmt"
)

// ProcessState describes the lifecycle state of one process runtime.
type ProcessState string

const (
	ProcessStateStopped  ProcessState = "stopped"
	ProcessStateStarting ProcessState = "starting"
	ProcessStateRunning  ProcessState = "running"
	ProcessStateStopping ProcessState = "stopping"
	ProcessStateExited   ProcessState = "exited"
	ProcessStateFailed   ProcessState = "failed"
)

var (
	// ErrInvalidProcessState reports an unknown source or target state.
	ErrInvalidProcessState = errors.New("invalid process state")
	// ErrInvalidStateTransition reports a transition that the lifecycle model forbids.
	ErrInvalidStateTransition = errors.New("invalid process state transition")
)

// IsValid reports whether state belongs to the lifecycle model.
func (state ProcessState) IsValid() bool {
	switch state {
	case ProcessStateStopped,
		ProcessStateStarting,
		ProcessStateRunning,
		ProcessStateStopping,
		ProcessStateExited,
		ProcessStateFailed:
		return true
	default:
		return false
	}
}

// IsActive reports whether a runtime may still own an operating-system process.
func (state ProcessState) IsActive() bool {
	switch state {
	case ProcessStateStarting, ProcessStateRunning, ProcessStateStopping:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether a runtime can no longer transition. A subsequent
// start attempt is represented by a new ProcessRuntime.
func (state ProcessState) IsTerminal() bool {
	switch state {
	case ProcessStateStopped, ProcessStateExited, ProcessStateFailed:
		return true
	default:
		return false
	}
}

// CanTransition reports whether one runtime may move directly between states.
func CanTransition(from ProcessState, to ProcessState) bool {
	switch from {
	case ProcessStateStarting:
		return to == ProcessStateRunning || to == ProcessStateStopping || to == ProcessStateFailed
	case ProcessStateRunning:
		return to == ProcessStateStopping || to == ProcessStateExited || to == ProcessStateFailed
	case ProcessStateStopping:
		return to == ProcessStateStopped || to == ProcessStateFailed
	default:
		return false
	}
}

// Transition moves a runtime to an allowed state without changing lifecycle
// metadata. The lifecycle service owns PID, timestamps, exit information, and
// termination reasons associated with the transition.
func (runtime *ProcessRuntime) Transition(next ProcessState) error {
	if !runtime.State.IsValid() {
		return fmt.Errorf("%w: current state %q", ErrInvalidProcessState, runtime.State)
	}
	if !next.IsValid() {
		return fmt.Errorf("%w: target state %q", ErrInvalidProcessState, next)
	}
	if !CanTransition(runtime.State, next) {
		return fmt.Errorf("%w: %s to %s", ErrInvalidStateTransition, runtime.State, next)
	}

	runtime.State = next
	return nil
}
