package supervision

import (
	"fmt"
	"time"
)

const DaemonRestartTerminationReason = "daemon-restart"

// Registry is the daemon-owned in-memory representation of persisted project
// definitions and their current or most recent process runtimes.
type Registry struct {
	Projects map[ProjectID]*RegisteredProject
}

// RegisteredProject groups one project definition with its process definitions.
type RegisteredProject struct {
	Definition ProjectDefinition
	Processes  map[ProcessID]*RegisteredProcess
}

// RegisteredProcess groups one process definition with its current or most
// recent runtime.
type RegisteredProcess struct {
	Definition ProcessDefinition
	Runtime    *ProcessRuntime
}

// NewRegistry returns an empty initialized registry.
func NewRegistry() *Registry {
	return &Registry{Projects: make(map[ProjectID]*RegisteredProject)}
}

// ReconcileAfterDaemonRestart marks every persisted active runtime as failed.
// Persisted PIDs remain diagnostic data and are never used to signal a process.
func (registry *Registry) ReconcileAfterDaemonRestart(at time.Time) (bool, error) {
	changed := false
	for projectID, project := range registry.Projects {
		if project == nil {
			return false, fmt.Errorf("project %q has no record", projectID)
		}
		for processID, process := range project.Processes {
			if process == nil {
				return false, fmt.Errorf("process %q in project %q has no record", processID, projectID)
			}
			if process.Runtime == nil {
				continue
			}
			if !process.Runtime.State.IsValid() {
				return false, fmt.Errorf("process %q in project %q has invalid runtime state %q", processID, projectID, process.Runtime.State)
			}
			if !process.Runtime.State.IsActive() {
				continue
			}
			if err := process.Runtime.Transition(ProcessStateFailed); err != nil {
				return false, fmt.Errorf("reconcile process %q in project %q: %w", processID, projectID, err)
			}
			finishedAt := at
			process.Runtime.FinishedAt = &finishedAt
			process.Runtime.TerminationReason = DaemonRestartTerminationReason
			changed = true
		}
	}

	return changed, nil
}
