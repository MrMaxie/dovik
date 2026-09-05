package mcpoperator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/MrMaxie/dovik/internal/supervision"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeClient struct {
	operatorclient.Client // Unimplemented methods must never be called by this surface.
	listProjects          func(context.Context) ([]supervision.ProjectDefinition, error)
	listProcesses         func(context.Context, supervision.ProjectID) ([]supervision.ProcessDefinition, error)
	logs                  func(context.Context, supervision.ProjectID, supervision.ProcessID, uint64, int) (supervision.OutputTail, error)
}

func (client *fakeClient) ListProjects(ctx context.Context) ([]supervision.ProjectDefinition, error) {
	return client.listProjects(ctx)
}

func (client *fakeClient) ListProcesses(ctx context.Context, id supervision.ProjectID) ([]supervision.ProcessDefinition, error) {
	return client.listProcesses(ctx, id)
}

func (client *fakeClient) Logs(ctx context.Context, project supervision.ProjectID, process supervision.ProcessID, after uint64, limit int) (supervision.OutputTail, error) {
	return client.logs(ctx, project, process, after, limit)
}

func testSession(t *testing.T, client operatorclient.Client) (*mcp.ClientSession, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := NewServer(client, io.Discard).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, ctx
}

func call(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestDiscoveryAndValidationBeforeIPC(t *testing.T) {
	session, ctx := testSession(t, &fakeClient{})
	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{"list_projects": true, "list_processes": true, "process_status": true, "process_logs": true, "process_start": false, "process_stop": false, "process_restart": false}
	if len(list.Tools) != len(expected) {
		t.Fatalf("tools = %d", len(list.Tools))
	}
	for _, tool := range list.Tools {
		readOnly, ok := expected[tool.Name]
		if !ok || tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("unexpected tool: %#v", tool)
		}
		annotations := tool.Annotations
		if annotations == nil || annotations.ReadOnlyHint != readOnly || annotations.OpenWorldHint == nil || *annotations.OpenWorldHint || annotations.DestructiveHint == nil || *annotations.DestructiveHint == readOnly || annotations.IdempotentHint {
			t.Fatalf("annotations for %s = %#v", tool.Name, annotations)
		}
	}
	for _, test := range []struct{ name, args string }{
		{"list_projects", `{"unexpected":true}`},
		{"list_processes", `{}`},
		{"list_processes", `{"projectId":" "}`},
		{"process_status", `{"projectId":"p"}`},
		{"process_start", `{"projectId":"p","processId":""}`},
		{"process_stop", `{"projectId":"p","processId":4}`},
		{"process_restart", `{"projectId":"p","processId":"x","command":"bad"}`},
		{"process_logs", `{"projectId":"p","processId":"x","limit":0}`},
		{"process_logs", `{"projectId":"p","processId":"x","limit":-1}`},
		{"process_logs", `{"projectId":"p","processId":"x","limit":1001}`},
		{"process_logs", `{"projectId":"p","processId":"x","limit":1.5}`},
		{"process_logs", `{"projectId":"p","processId":"x","afterSequence":-1}`},
	} {
		t.Run(test.name+test.args, func(t *testing.T) {
			result := call(t, ctx, session, test.name, json.RawMessage(test.args))
			if !result.IsError {
				t.Fatalf("invalid arguments accepted: %#v", result)
			}
		})
	}
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "process_add", Arguments: map[string]any{}}); err == nil {
		t.Fatal("unknown tool accepted")
	}
}

func TestStructuredOutputAndDefinitionPrivacy(t *testing.T) {
	session, ctx := testSession(t, &fakeClient{
		listProjects: func(context.Context) ([]supervision.ProjectDefinition, error) { return nil, nil },
		listProcesses: func(_ context.Context, id supervision.ProjectID) ([]supervision.ProcessDefinition, error) {
			if id != "p" {
				t.Errorf("project = %q", id)
			}
			return []supervision.ProcessDefinition{{ProjectID: "p", ID: "worker", Command: "worker", EnvironmentOverrides: map[string]string{"SECRET": "private-value"}}}, nil
		},
	})
	projects := call(t, ctx, session, "list_projects", map[string]any{})
	if projects.IsError {
		t.Fatalf("projects: %#v", projects)
	}
	data, _ := json.Marshal(projects.StructuredContent)
	if string(data) != `{"projects":[]}` {
		t.Fatalf("empty projects: %s", data)
	}
	processes := call(t, ctx, session, "list_processes", map[string]any{"projectId": "p"})
	if processes.IsError {
		t.Fatalf("processes: %#v", processes)
	}
	data, _ = json.Marshal(processes)
	if strings.Contains(string(data), "private-value") || strings.Contains(strings.ToLower(string(data)), "environment") || !strings.Contains(string(data), "worker") {
		t.Fatalf("definition output: %s", data)
	}
	if len(processes.Content) == 0 || processes.StructuredContent == nil {
		t.Fatal("missing structured output or text fallback")
	}
}

func TestLogDefaultsCursorAndAttribution(t *testing.T) {
	for _, test := range []struct {
		args  map[string]any
		after uint64
		limit int
	}{
		{map[string]any{"projectId": "p", "processId": "worker"}, 0, 100},
		{map[string]any{"projectId": "p", "processId": "worker", "afterSequence": 41, "limit": 1}, 41, 1},
	} {
		session, ctx := testSession(t, &fakeClient{logs: func(_ context.Context, project supervision.ProjectID, process supervision.ProcessID, after uint64, limit int) (supervision.OutputTail, error) {
			if project != "p" || process != "worker" || after != test.after || limit != test.limit {
				t.Errorf("unexpected logs arguments: %s/%s %d %d", project, process, after, limit)
			}
			return supervision.OutputTail{Truncated: true, Events: []supervision.OutputEvent{{Sequence: 42, CapturedAt: time.Unix(1, 0).UTC(), Stream: supervision.OutputStreamStderr, Data: []byte("output\n")}}}, nil
		}})
		result := call(t, ctx, session, "process_logs", test.args)
		if result.IsError {
			t.Fatalf("logs: %#v", result)
		}
		data, _ := json.Marshal(result.StructuredContent)
		var tail logsResult
		if err := json.Unmarshal(data, &tail); err != nil {
			t.Fatal(err)
		}
		if !tail.Truncated || len(tail.Events) != 1 || tail.Events[0].Sequence != 42 || tail.Events[0].Stream != supervision.OutputStreamStderr || tail.Events[0].Data != "output\n" {
			t.Fatalf("logs = %s", data)
		}
	}
}

func TestToolErrorsDoNotExposePrivateDiagnostics(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{errors.New("private-value pipe path"), "connection_error"},
		{&control.ProtocolError{Code: "operation_failed", Message: "private-value"}, "operation_failed"},
		{&control.ProtocolError{Code: "incompatible_version", Message: "private-value"}, "incompatible_version"},
		{&control.ProtocolError{Code: "internal_error", Message: "private-value"}, "daemon_error"},
		{context.DeadlineExceeded, "timeout"},
		{context.Canceled, "canceled"},
	} {
		t.Run(test.code, func(t *testing.T) {
			session, ctx := testSession(t, &fakeClient{listProjects: func(context.Context) ([]supervision.ProjectDefinition, error) { return nil, test.err }})
			result := call(t, ctx, session, "list_projects", map[string]any{})
			data, _ := json.Marshal(result)
			if !result.IsError || !strings.Contains(string(data), test.code+":") || strings.Contains(string(data), "private-value") {
				t.Fatalf("error = %s", data)
			}
		})
	}
}

func TestRequestCancellationReachesDaemonCall(t *testing.T) {
	entered, finished := make(chan struct{}), make(chan struct{})
	session, ctx := testSession(t, &fakeClient{listProjects: func(ctx context.Context) ([]supervision.ProjectDefinition, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > callTimeout {
			t.Error("missing bounded deadline")
		}
		close(entered)
		<-ctx.Done()
		close(finished)
		return nil, ctx.Err()
	}})
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := session.CallTool(requestCtx, &mcp.CallToolParams{Name: "list_projects", Arguments: map[string]any{}})
		done <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("handler did not start")
	}
	cancel()
	select {
	case <-finished:
	case <-ctx.Done():
		t.Fatal("daemon call was not canceled")
	}
	if err := <-done; err == nil {
		t.Fatal("canceled call succeeded")
	}
}
