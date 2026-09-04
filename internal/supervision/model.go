package supervision

import "time"

// ProjectID identifies a registered local project.
type ProjectID string

// ProcessID identifies a process definition within a project.
type ProcessID string

// RuntimeInstanceID identifies one process start attempt.
type RuntimeInstanceID string

// ProjectDefinition describes a local project owned by the daemon registry.
type ProjectDefinition struct {
	ID            ProjectID
	RootDirectory string
}

// ProcessDefinition describes a process owned by one registered project.
type ProcessDefinition struct {
	ProjectID            ProjectID
	ID                   ProcessID
	Command              string
	Arguments            []string
	WorkingDirectory     string
	EnvironmentOverrides map[string]string
}

// ProcessRuntime records the current or most recent start attempt for a process
// definition. PID is diagnostic data and must not be treated as process authority
// after a daemon restart.
type ProcessRuntime struct {
	ProjectID         ProjectID
	ProcessID         ProcessID
	InstanceID        RuntimeInstanceID
	PID               *int
	State             ProcessState
	StartedAt         *time.Time
	FinishedAt        *time.Time
	ExitCode          *int
	TerminationReason string
}

// NewProcessRuntime creates the model for a new start attempt.
func NewProcessRuntime(projectID ProjectID, processID ProcessID, instanceID RuntimeInstanceID) ProcessRuntime {
	return ProcessRuntime{
		ProjectID:  projectID,
		ProcessID:  processID,
		InstanceID: instanceID,
		State:      ProcessStateStarting,
	}
}
