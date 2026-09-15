package control

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MrMaxie/dovik/internal/supervision"
)

const ProtocolVersion = 1

type Operation string

const (
	OperationIdentity       Operation = "identity"
	OperationGHExecute      Operation = "identity.execute"
	OperationAgentRun       Operation = "identity.agent-run"
	OperationProjectAdd     Operation = "project.add"
	OperationProjectRemove  Operation = "project.remove"
	OperationProjectList    Operation = "project.list"
	OperationProcessAdd     Operation = "process.add"
	OperationProcessRemove  Operation = "process.remove"
	OperationProcessList    Operation = "process.list"
	OperationProcessStart   Operation = "process.start"
	OperationProcessStop    Operation = "process.stop"
	OperationProcessRestart Operation = "process.restart"
	OperationProcessStatus  Operation = "process.status"
	OperationProcessLogs    Operation = "process.logs"
)

type Request struct {
	Version   int             `json:"version"`
	ID        string          `json:"id"`
	Operation Operation       `json:"operation"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ProtocolError  `json:"error,omitempty"`
}

type ProtocolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (protocolError *ProtocolError) Error() string {
	return protocolError.Code + ": " + protocolError.Message
}

type projectInput struct {
	ID            supervision.ProjectID `json:"id"`
	RootDirectory string                `json:"rootDirectory"`
}

type projectReference struct {
	ProjectID supervision.ProjectID `json:"projectId"`
}

type processInput struct {
	ProjectID            supervision.ProjectID `json:"projectId"`
	ID                   supervision.ProcessID `json:"id"`
	Command              string                `json:"command"`
	Arguments            []string              `json:"arguments,omitempty"`
	WorkingDirectory     string                `json:"workingDirectory,omitempty"`
	EnvironmentOverrides map[string]string     `json:"environmentOverrides,omitempty"`
}

type processReference struct {
	ProjectID supervision.ProjectID `json:"projectId"`
	ProcessID supervision.ProcessID `json:"processId"`
}

type logsInput struct {
	ProjectID     supervision.ProjectID `json:"projectId"`
	ProcessID     supervision.ProcessID `json:"processId"`
	AfterSequence uint64                `json:"afterSequence,omitempty"`
	Limit         int                   `json:"limit"`
}

type projectView struct {
	ID            supervision.ProjectID `json:"id"`
	RootDirectory string                `json:"rootDirectory"`
}

type processView struct {
	ProjectID        supervision.ProjectID `json:"projectId"`
	ID               supervision.ProcessID `json:"id"`
	Command          string                `json:"command"`
	Arguments        []string              `json:"arguments,omitempty"`
	WorkingDirectory string                `json:"workingDirectory,omitempty"`
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

type statusView struct {
	Exists  bool         `json:"exists"`
	Runtime *runtimeView `json:"runtime,omitempty"`
}

type outputEventView struct {
	Sequence   uint64                   `json:"sequence"`
	CapturedAt time.Time                `json:"capturedAt"`
	Stream     supervision.OutputStream `json:"stream"`
	Data       []byte                   `json:"data"`
}

type outputTailView struct {
	Events    []outputEventView `json:"events"`
	Truncated bool              `json:"truncated"`
}

func projectToView(project supervision.ProjectDefinition) projectView {
	return projectView{ID: project.ID, RootDirectory: project.RootDirectory}
}

func processToView(process supervision.ProcessDefinition) processView {
	return processView{
		ProjectID:        process.ProjectID,
		ID:               process.ID,
		Command:          process.Command,
		Arguments:        append([]string(nil), process.Arguments...),
		WorkingDirectory: process.WorkingDirectory,
	}
}

func runtimeToView(runtimeState supervision.ProcessRuntime) runtimeView {
	return runtimeView{
		ProjectID:         runtimeState.ProjectID,
		ProcessID:         runtimeState.ProcessID,
		InstanceID:        runtimeState.InstanceID,
		PID:               runtimeState.PID,
		State:             runtimeState.State,
		StartedAt:         runtimeState.StartedAt,
		FinishedAt:        runtimeState.FinishedAt,
		ExitCode:          runtimeState.ExitCode,
		TerminationReason: runtimeState.TerminationReason,
	}
}

func marshalResult(value any) (json.RawMessage, error) {
	result, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode result: %w", err)
	}
	return result, nil
}
