package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/MrMaxie/dovik/internal/supervision"
)

// Client is a versioned client for the local daemon protocol.
type Client struct {
	endpoint string
	dial     func(context.Context, string) (net.Conn, error)
	nextID   atomic.Uint64
}

func NewClient(endpoint string) *Client {
	return &Client{endpoint: endpoint, dial: DialLocal}
}

func (client *Client) AddProject(ctx context.Context, definition supervision.ProjectDefinition) (supervision.ProjectDefinition, error) {
	var view projectView
	err := client.call(ctx, OperationProjectAdd, projectInput{ID: definition.ID, RootDirectory: definition.RootDirectory}, &view)
	return supervision.ProjectDefinition{ID: view.ID, RootDirectory: view.RootDirectory}, err
}

func (client *Client) RemoveProject(ctx context.Context, projectID supervision.ProjectID) error {
	return client.call(ctx, OperationProjectRemove, projectReference{ProjectID: projectID}, nil)
}

func (client *Client) ListProjects(ctx context.Context) ([]supervision.ProjectDefinition, error) {
	var views []projectView
	if err := client.call(ctx, OperationProjectList, struct{}{}, &views); err != nil {
		return nil, err
	}
	projects := make([]supervision.ProjectDefinition, 0, len(views))
	for _, view := range views {
		projects = append(projects, supervision.ProjectDefinition{ID: view.ID, RootDirectory: view.RootDirectory})
	}
	return projects, nil
}

func (client *Client) AddProcess(ctx context.Context, definition supervision.ProcessDefinition) (supervision.ProcessDefinition, error) {
	var view processView
	err := client.call(ctx, OperationProcessAdd, processInput{
		ProjectID:            definition.ProjectID,
		ID:                   definition.ID,
		Command:              definition.Command,
		Arguments:            definition.Arguments,
		WorkingDirectory:     definition.WorkingDirectory,
		EnvironmentOverrides: definition.EnvironmentOverrides,
	}, &view)
	return processFromView(view), err
}

func (client *Client) RemoveProcess(ctx context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) error {
	return client.call(ctx, OperationProcessRemove, processReference{ProjectID: projectID, ProcessID: processID}, nil)
}

func (client *Client) ListProcesses(ctx context.Context, projectID supervision.ProjectID) ([]supervision.ProcessDefinition, error) {
	var views []processView
	if err := client.call(ctx, OperationProcessList, projectReference{ProjectID: projectID}, &views); err != nil {
		return nil, err
	}
	processes := make([]supervision.ProcessDefinition, 0, len(views))
	for _, view := range views {
		processes = append(processes, processFromView(view))
	}
	return processes, nil
}

func (client *Client) Start(ctx context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, error) {
	return client.runtimeOperation(ctx, OperationProcessStart, projectID, processID)
}

func (client *Client) Stop(ctx context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, error) {
	return client.runtimeOperation(ctx, OperationProcessStop, projectID, processID)
}

func (client *Client) Restart(ctx context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, error) {
	return client.runtimeOperation(ctx, OperationProcessRestart, projectID, processID)
}

func (client *Client) Status(ctx context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, bool, error) {
	var view statusView
	if err := client.call(ctx, OperationProcessStatus, processReference{ProjectID: projectID, ProcessID: processID}, &view); err != nil {
		return supervision.ProcessRuntime{}, false, err
	}
	if !view.Exists || view.Runtime == nil {
		return supervision.ProcessRuntime{}, false, nil
	}
	return runtimeFromView(*view.Runtime), true, nil
}

func (client *Client) Logs(ctx context.Context, projectID supervision.ProjectID, processID supervision.ProcessID, afterSequence uint64, limit int) (supervision.OutputTail, error) {
	var view outputTailView
	if err := client.call(ctx, OperationProcessLogs, logsInput{
		ProjectID: projectID, ProcessID: processID, AfterSequence: afterSequence, Limit: limit,
	}, &view); err != nil {
		return supervision.OutputTail{}, err
	}
	tail := supervision.OutputTail{Truncated: view.Truncated, Events: make([]supervision.OutputEvent, 0, len(view.Events))}
	for _, event := range view.Events {
		tail.Events = append(tail.Events, supervision.OutputEvent(event))
	}
	return tail, nil
}

func (client *Client) runtimeOperation(ctx context.Context, operation Operation, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, error) {
	var view runtimeView
	err := client.call(ctx, operation, processReference{ProjectID: projectID, ProcessID: processID}, &view)
	return runtimeFromView(view), err
}

func (client *Client) call(ctx context.Context, operation Operation, payload any, result any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request payload: %w", err)
	}
	requestID := strconv.FormatUint(client.nextID.Add(1), 10)
	connection, err := client.dial(ctx, client.endpoint)
	if err != nil {
		return err
	}
	defer connection.Close()
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.Close()
		case <-finished:
		}
	}()
	request := Request{Version: ProtocolVersion, ID: requestID, Operation: operation, Payload: payloadJSON}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		return fmt.Errorf("write control request: %w", err)
	}
	var response Response
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		return fmt.Errorf("read control response: %w", err)
	}
	if response.Version != ProtocolVersion {
		return fmt.Errorf("daemon returned incompatible protocol version %d", response.Version)
	}
	if response.ID != requestID {
		return fmt.Errorf("daemon response ID %q does not match request %q", response.ID, requestID)
	}
	if response.Error != nil {
		return response.Error
	}
	if result == nil || len(response.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("decode control result: %w", err)
	}
	return nil
}

func processFromView(view processView) supervision.ProcessDefinition {
	return supervision.ProcessDefinition{
		ProjectID:        view.ProjectID,
		ID:               view.ID,
		Command:          view.Command,
		Arguments:        append([]string(nil), view.Arguments...),
		WorkingDirectory: view.WorkingDirectory,
	}
}

func runtimeFromView(view runtimeView) supervision.ProcessRuntime {
	return supervision.ProcessRuntime{
		ProjectID:         view.ProjectID,
		ProcessID:         view.ProcessID,
		InstanceID:        view.InstanceID,
		PID:               view.PID,
		State:             view.State,
		StartedAt:         view.StartedAt,
		FinishedAt:        view.FinishedAt,
		ExitCode:          view.ExitCode,
		TerminationReason: view.TerminationReason,
	}
}
