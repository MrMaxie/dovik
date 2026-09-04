package supervision

import (
	"errors"
	"testing"
)

func TestNewProcessRuntimeStartsOneAttempt(t *testing.T) {
	runtime := NewProcessRuntime("project", "api", "runtime-1")

	if runtime.ProjectID != "project" {
		t.Fatalf("ProjectID = %q, want %q", runtime.ProjectID, ProjectID("project"))
	}
	if runtime.ProcessID != "api" {
		t.Fatalf("ProcessID = %q, want %q", runtime.ProcessID, ProcessID("api"))
	}
	if runtime.InstanceID != "runtime-1" {
		t.Fatalf("InstanceID = %q, want %q", runtime.InstanceID, RuntimeInstanceID("runtime-1"))
	}
	if runtime.State != ProcessStateStarting {
		t.Fatalf("State = %q, want %q", runtime.State, ProcessStateStarting)
	}
}

func TestProcessStateClassification(t *testing.T) {
	tests := []struct {
		name     string
		state    ProcessState
		valid    bool
		active   bool
		terminal bool
	}{
		{name: "stopped", state: ProcessStateStopped, valid: true, terminal: true},
		{name: "starting", state: ProcessStateStarting, valid: true, active: true},
		{name: "running", state: ProcessStateRunning, valid: true, active: true},
		{name: "stopping", state: ProcessStateStopping, valid: true, active: true},
		{name: "exited", state: ProcessStateExited, valid: true, terminal: true},
		{name: "failed", state: ProcessStateFailed, valid: true, terminal: true},
		{name: "unknown", state: ProcessState("unknown")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.state.IsValid(); got != test.valid {
				t.Errorf("IsValid() = %t, want %t", got, test.valid)
			}
			if got := test.state.IsActive(); got != test.active {
				t.Errorf("IsActive() = %t, want %t", got, test.active)
			}
			if got := test.state.IsTerminal(); got != test.terminal {
				t.Errorf("IsTerminal() = %t, want %t", got, test.terminal)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	states := []ProcessState{
		ProcessStateStopped,
		ProcessStateStarting,
		ProcessStateRunning,
		ProcessStateStopping,
		ProcessStateExited,
		ProcessStateFailed,
	}
	allowed := map[[2]ProcessState]bool{
		{ProcessStateStarting, ProcessStateRunning}:  true,
		{ProcessStateStarting, ProcessStateStopping}: true,
		{ProcessStateStarting, ProcessStateFailed}:   true,
		{ProcessStateRunning, ProcessStateStopping}:  true,
		{ProcessStateRunning, ProcessStateExited}:    true,
		{ProcessStateRunning, ProcessStateFailed}:    true,
		{ProcessStateStopping, ProcessStateStopped}:  true,
		{ProcessStateStopping, ProcessStateFailed}:   true,
	}

	for _, from := range states {
		for _, to := range states {
			want := allowed[[2]ProcessState{from, to}]
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%q, %q) = %t, want %t", from, to, got, want)
			}
		}
	}
}

func TestProcessRuntimeTransition(t *testing.T) {
	runtime := NewProcessRuntime("project", "api", "runtime-1")

	if err := runtime.Transition(ProcessStateRunning); err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
	if runtime.State != ProcessStateRunning {
		t.Fatalf("State = %q, want %q", runtime.State, ProcessStateRunning)
	}
}

func TestProcessRuntimeTransitionRejectsUnknownState(t *testing.T) {
	runtime := NewProcessRuntime("project", "api", "runtime-1")

	err := runtime.Transition(ProcessState("unknown"))
	if !errors.Is(err, ErrInvalidProcessState) {
		t.Fatalf("Transition() error = %v, want ErrInvalidProcessState", err)
	}
	if runtime.State != ProcessStateStarting {
		t.Fatalf("State = %q after rejected transition, want %q", runtime.State, ProcessStateStarting)
	}
}

func TestProcessRuntimeTransitionRejectsInvalidPath(t *testing.T) {
	runtime := NewProcessRuntime("project", "api", "runtime-1")

	err := runtime.Transition(ProcessStateExited)
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("Transition() error = %v, want ErrInvalidStateTransition", err)
	}
	if runtime.State != ProcessStateStarting {
		t.Fatalf("State = %q after rejected transition, want %q", runtime.State, ProcessStateStarting)
	}
}

func TestProcessRuntimeTransitionRejectsTerminalRuntime(t *testing.T) {
	runtime := ProcessRuntime{State: ProcessStateExited}

	err := runtime.Transition(ProcessStateStarting)
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("Transition() error = %v, want ErrInvalidStateTransition", err)
	}
	if runtime.State != ProcessStateExited {
		t.Fatalf("State = %q after rejected transition, want %q", runtime.State, ProcessStateExited)
	}
}
