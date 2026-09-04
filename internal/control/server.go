package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/MrMaxie/dovik/internal/supervision"
)

const maximumRequestBytes = 1024 * 1024

// Server exposes one lifecycle manager through the local control protocol.
type Server struct {
	manager *supervision.LifecycleManager
}

func NewServer(manager *supervision.LifecycleManager) *Server {
	return &Server{manager: manager}
}

// Serve accepts local client connections until the context is canceled.
func (server *Server) Serve(ctx context.Context, listener net.Listener) error {
	var connections sync.WaitGroup
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	defer connections.Wait()
	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept control connection: %w", err)
		}
		connections.Add(1)
		go func() {
			defer connections.Done()
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
			server.handleConnection(ctx, connection)
		}()
	}
}

func (server *Server) handleConnection(ctx context.Context, connection io.ReadWriter) {
	decoder := json.NewDecoder(io.LimitReader(connection, maximumRequestBytes))
	decoder.DisallowUnknownFields()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		server.writeResponse(connection, Response{
			Version: ProtocolVersion,
			Error:   &ProtocolError{Code: "invalid_request", Message: err.Error()},
		})
		return
	}
	response := server.handleRequest(ctx, request)
	server.writeResponse(connection, response)
}

func (server *Server) handleRequest(ctx context.Context, request Request) Response {
	response := Response{Version: ProtocolVersion, ID: request.ID}
	if request.Version != ProtocolVersion {
		response.Error = &ProtocolError{Code: "incompatible_version", Message: fmt.Sprintf("protocol version %d is not supported", request.Version)}
		return response
	}
	if request.ID == "" {
		response.Error = &ProtocolError{Code: "invalid_request", Message: "request ID is required"}
		return response
	}

	result, err := server.execute(ctx, request)
	if err != nil {
		response.Error = &ProtocolError{Code: "operation_failed", Message: err.Error()}
		return response
	}
	response.Result, err = marshalResult(result)
	if err != nil {
		response.Error = &ProtocolError{Code: "internal_error", Message: "response could not be encoded"}
	}
	return response
}

func (server *Server) execute(ctx context.Context, request Request) (any, error) {
	switch request.Operation {
	case OperationProjectAdd:
		var input projectInput
		if err := decodePayload(request.Payload, &input); err != nil {
			return nil, err
		}
		project, err := server.manager.AddProject(supervision.ProjectDefinition{ID: input.ID, RootDirectory: input.RootDirectory})
		return projectToView(project), err
	case OperationProjectRemove:
		var input projectReference
		if err := decodePayload(request.Payload, &input); err != nil {
			return nil, err
		}
		return struct{}{}, server.manager.RemoveProject(input.ProjectID)
	case OperationProjectList:
		projects := server.manager.ListProjects()
		views := make([]projectView, 0, len(projects))
		for _, project := range projects {
			views = append(views, projectToView(project))
		}
		return views, nil
	case OperationProcessAdd:
		var input processInput
		if err := decodePayload(request.Payload, &input); err != nil {
			return nil, err
		}
		process, err := server.manager.AddProcess(supervision.ProcessDefinition{
			ProjectID:            input.ProjectID,
			ID:                   input.ID,
			Command:              input.Command,
			Arguments:            input.Arguments,
			WorkingDirectory:     input.WorkingDirectory,
			EnvironmentOverrides: input.EnvironmentOverrides,
		})
		return processToView(process), err
	case OperationProcessRemove:
		input, err := decodeProcessReference(request.Payload)
		if err != nil {
			return nil, err
		}
		return struct{}{}, server.manager.RemoveProcess(input.ProjectID, input.ProcessID)
	case OperationProcessList:
		var input projectReference
		if err := decodePayload(request.Payload, &input); err != nil {
			return nil, err
		}
		processes, err := server.manager.ListProcesses(input.ProjectID)
		if err != nil {
			return nil, err
		}
		views := make([]processView, 0, len(processes))
		for _, process := range processes {
			views = append(views, processToView(process))
		}
		return views, nil
	case OperationProcessStart:
		input, err := decodeProcessReference(request.Payload)
		if err != nil {
			return nil, err
		}
		runtimeState, err := server.manager.Start(input.ProjectID, input.ProcessID)
		return runtimeToView(runtimeState), err
	case OperationProcessStop:
		input, err := decodeProcessReference(request.Payload)
		if err != nil {
			return nil, err
		}
		runtimeState, err := server.manager.Stop(ctx, input.ProjectID, input.ProcessID)
		return runtimeToView(runtimeState), err
	case OperationProcessRestart:
		input, err := decodeProcessReference(request.Payload)
		if err != nil {
			return nil, err
		}
		runtimeState, err := server.manager.Restart(ctx, input.ProjectID, input.ProcessID)
		return runtimeToView(runtimeState), err
	case OperationProcessStatus:
		input, err := decodeProcessReference(request.Payload)
		if err != nil {
			return nil, err
		}
		runtimeState, exists, err := server.manager.Status(input.ProjectID, input.ProcessID)
		if err != nil || !exists {
			return statusView{Exists: exists}, err
		}
		view := runtimeToView(runtimeState)
		return statusView{Exists: true, Runtime: &view}, nil
	case OperationProcessLogs:
		var input logsInput
		if err := decodePayload(request.Payload, &input); err != nil {
			return nil, err
		}
		tail, err := server.manager.Logs(input.ProjectID, input.ProcessID, input.AfterSequence, input.Limit)
		if err != nil {
			return nil, err
		}
		view := outputTailView{Truncated: tail.Truncated, Events: make([]outputEventView, 0, len(tail.Events))}
		for _, event := range tail.Events {
			view.Events = append(view.Events, outputEventView(event))
		}
		return view, nil
	default:
		return nil, fmt.Errorf("operation %q is not supported", request.Operation)
	}
}

func decodeProcessReference(payload json.RawMessage) (processReference, error) {
	var input processReference
	err := decodePayload(payload, &input)
	return input, err
}

func decodePayload(payload json.RawMessage, target any) error {
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid operation payload: %w", err)
	}
	return nil
}

func (server *Server) writeResponse(writer io.Writer, response Response) {
	_ = json.NewEncoder(writer).Encode(response)
}
