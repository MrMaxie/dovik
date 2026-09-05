package mcpoperator

import (
	"time"

	"github.com/MrMaxie/dovik/internal/supervision"
)

// Explicit views keep private definition fields out of MCP results and schemas.
type projectView struct {
	ID            supervision.ProjectID `json:"id"`
	RootDirectory string                `json:"rootDirectory"`
}

type projectsResult struct {
	Projects []projectView `json:"projects"`
}

type processView struct {
	ProjectID        supervision.ProjectID `json:"projectId"`
	ID               supervision.ProcessID `json:"id"`
	Command          string                `json:"command"`
	Arguments        []string              `json:"arguments,omitempty"`
	WorkingDirectory string                `json:"workingDirectory,omitempty"`
}

type processesResult struct {
	Processes []processView `json:"processes"`
}

type runtimeView struct {
	ProjectID         supervision.ProjectID         `json:"projectId"`
	ProcessID         supervision.ProcessID         `json:"processId"`
	InstanceID        supervision.RuntimeInstanceID `json:"instanceId"`
	PID               *int                          `json:"pid,omitempty"`
	State             supervision.ProcessState      `json:"state"`
	StartedAt         *time.Time                    `json:"startedAt,omitempty"`
	FinishedAt        *time.Time                    `json:"finishedAt,omitempty"`
	ExitCode          *int                          `json:"exitCode,omitempty"`
	TerminationReason string                        `json:"terminationReason,omitempty"`
}

type statusResult struct {
	Exists  bool         `json:"exists"`
	Runtime *runtimeView `json:"runtime,omitempty"`
}

type outputEventView struct {
	Sequence   uint64                   `json:"sequence"`
	CapturedAt time.Time                `json:"capturedAt"`
	Stream     supervision.OutputStream `json:"stream"`
	Data       string                   `json:"data"`
}

type logsResult struct {
	Events    []outputEventView `json:"events"`
	Truncated bool              `json:"truncated"`
}

func runtimeResult(runtime supervision.ProcessRuntime) runtimeView {
	return runtimeView{
		ProjectID: runtime.ProjectID, ProcessID: runtime.ProcessID, InstanceID: runtime.InstanceID,
		PID: runtime.PID, State: runtime.State, StartedAt: runtime.StartedAt, FinishedAt: runtime.FinishedAt,
		ExitCode: runtime.ExitCode, TerminationReason: runtime.TerminationReason,
	}
}
