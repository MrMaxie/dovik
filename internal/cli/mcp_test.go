package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/supervision"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPStdioLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "dovik")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../cmd/dovik")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v: %s", err, output)
	}
	endpoint, stopServer := startCLITestServer(t)
	defer stopServer()
	daemon := control.NewClient(endpoint)
	if _, err := daemon.AddProject(ctx, supervision.ProjectDefinition{ID: "mcp", RootDirectory: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if _, err := daemon.AddProcess(ctx, supervision.ProcessDefinition{
		ProjectID: "mcp", ID: "worker", Command: os.Args[0], Arguments: []string{"-test.run=^TestMCPFixtureProcess$"},
		EnvironmentOverrides: map[string]string{"DOVIK_MCP_FIXTURE": "1", "SECRET": "private-fixture-value"},
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = daemon.Stop(cleanupCtx, "mcp", "worker")
	}()

	var stderr bytes.Buffer
	command := exec.CommandContext(ctx, binary, "--endpoint", endpoint, "mcp")
	command.Stderr = &stderr
	session, err := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("initialize stdio: %v", err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 7 {
		t.Fatalf("discover tools: %#v, %v", tools, err)
	}
	args := map[string]any{"projectId": "mcp", "processId": "worker"}
	invoke := func(name string, args any) json.RawMessage {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if result.IsError {
			t.Fatalf("%s: %#v", name, result.Content)
		}
		data, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "private-fixture-value") || strings.Contains(string(data), "environmentOverrides") {
			t.Fatalf("private definition data in %s", name)
		}
		return data
	}
	projects := invoke("list_projects", map[string]any{})
	if !strings.Contains(string(projects), `"id":"mcp"`) {
		t.Fatalf("projects = %s", projects)
	}
	processes := invoke("list_processes", map[string]any{"projectId": "mcp"})
	if !strings.Contains(string(processes), `"id":"worker"`) {
		t.Fatalf("processes = %s", processes)
	}
	initial := invoke("process_status", args)
	if string(initial) != `{"exists":false}` {
		t.Fatalf("initial status = %s", initial)
	}
	started := invoke("process_start", args)
	var firstRuntime runtimeJSON
	if err := json.Unmarshal(started, &firstRuntime); err != nil {
		t.Fatal(err)
	}
	if firstRuntime.State != "running" || firstRuntime.InstanceID == "" {
		t.Fatalf("start = %s", started)
	}
	status := invoke("process_status", args)
	if !strings.Contains(string(status), `"exists":true`) || !strings.Contains(string(status), firstRuntime.InstanceID) {
		t.Fatalf("status = %s", status)
	}

	var tail outputTailJSON
	logDeadline := time.Now().Add(5 * time.Second)
	for {
		data := invoke("process_logs", args)
		if err := json.Unmarshal(data, &tail); err != nil {
			t.Fatal(err)
		}
		streams := map[string]bool{}
		for _, event := range tail.Events {
			streams[event.Stream] = true
		}
		if streams["stdout"] && streams["stderr"] {
			break
		}
		if time.Now().After(logDeadline) {
			t.Fatalf("missing output streams: %s", data)
		}
		time.Sleep(20 * time.Millisecond)
	}
	bounded := invoke("process_logs", map[string]any{"projectId": "mcp", "processId": "worker", "limit": 1})
	var single outputTailJSON
	if err := json.Unmarshal(bounded, &single); err != nil {
		t.Fatal(err)
	}
	if len(single.Events) != 1 {
		t.Fatalf("bounded logs = %s", bounded)
	}
	cursor := invoke("process_logs", map[string]any{"projectId": "mcp", "processId": "worker", "afterSequence": single.Events[0].Sequence})
	var after outputTailJSON
	if err := json.Unmarshal(cursor, &after); err != nil {
		t.Fatal(err)
	}
	if len(after.Events) != 0 {
		t.Fatalf("cursor replayed events: %s", cursor)
	}
	stopped := invoke("process_stop", args)
	if !strings.Contains(string(stopped), `"state":"stopped"`) {
		t.Fatalf("stop = %s", stopped)
	}
	restarted := invoke("process_restart", args)
	var lastRuntime runtimeJSON
	if err := json.Unmarshal(restarted, &lastRuntime); err != nil {
		t.Fatal(err)
	}
	if lastRuntime.State != "running" || lastRuntime.InstanceID == firstRuntime.InstanceID {
		t.Fatalf("restart = %s", restarted)
	}

	missing, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "process_start", Arguments: map[string]any{"projectId": "mcp", "processId": "missing"}})
	if err != nil || !missing.IsError {
		t.Fatalf("missing definition = %#v, %v", missing, err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("close MCP: %v", err)
	}
	if command.ProcessState == nil || !command.ProcessState.Success() {
		t.Fatalf("stdio did not exit cleanly: %v; %s", command.ProcessState, stderr.String())
	}
	remaining, exists, err := daemon.Status(ctx, "mcp", "worker")
	if err != nil || !exists || remaining.State != supervision.ProcessStateRunning || string(remaining.InstanceID) != lastRuntime.InstanceID {
		t.Fatalf("MCP exit affected daemon ownership: %#v, %v, %v", remaining, exists, err)
	}
}

func TestMCPFixtureProcess(t *testing.T) {
	if os.Getenv("DOVIK_MCP_FIXTURE") != "1" {
		return
	}
	_, _ = os.Stdout.WriteString("stdout-ready\n")
	_, _ = os.Stderr.WriteString("stderr-ready\n")
	for {
		time.Sleep(time.Second)
	}
}

func TestMCPCommandRejectsArgumentsAndJSON(t *testing.T) {
	for _, args := range [][]string{{"--json", "mcp"}, {"mcp", "extra"}, {"mcp", "--json"}} {
		var stdout, stderr bytes.Buffer
		if code := RunIO(context.Background(), args, strings.NewReader(""), &stdout, &stderr); code != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("args %v: code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestMCPDiscoveryWithoutDaemonAndSessionCancellation(t *testing.T) {
	endpoint, stopServer := startCLITestServer(t)
	stopServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverInput, clientOutput := io.Pipe()
	clientInput, serverOutput := io.Pipe()
	defer clientOutput.Close()
	defer clientInput.Close()
	done := make(chan int, 1)
	go func() {
		done <- RunIO(ctx, []string{"--endpoint", endpoint, "mcp"}, serverInput, serverOutput, io.Discard)
	}()
	clientCtx, clientCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer clientCancel()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(clientCtx, &mcp.IOTransport{Reader: clientInput, Writer: clientOutput}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	list, err := session.ListTools(clientCtx, nil)
	if err != nil || len(list.Tools) != 7 {
		t.Fatalf("offline discovery: %#v %v", list, err)
	}
	result, err := session.CallTool(clientCtx, &mcp.CallToolParams{Name: "list_projects", Arguments: map[string]any{}})
	if err != nil || !result.IsError {
		t.Fatalf("offline call: %#v %v", result, err)
	}
	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("canceled session exit = %d", code)
		}
	case <-clientCtx.Done():
		t.Fatal("MCP session did not exit on cancellation")
	}
	_ = serverOutput.Close()
}
