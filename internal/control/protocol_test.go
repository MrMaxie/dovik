package control

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/MrMaxie/dovik/internal/supervision"
)

const controlHelperEnvironment = "DOVIK_CONTROL_HELPER"

func TestClientServerLifecycleRoundTrip(t *testing.T) {
	client, stopServer := startTestServer(t)
	defer stopServer()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root := t.TempDir()

	if _, err := client.AddProject(ctx, supervision.ProjectDefinition{ID: "project", RootDirectory: root}); err != nil {
		t.Fatalf("AddProject() error = %v", err)
	}
	process, err := client.AddProcess(ctx, supervision.ProcessDefinition{
		ProjectID:            "project",
		ID:                   "api",
		Command:              os.Args[0],
		Arguments:            []string{"-test.run=TestControlHelperProcess"},
		EnvironmentOverrides: map[string]string{controlHelperEnvironment: "1", "SECRET": "private"},
	})
	if err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	if process.EnvironmentOverrides != nil {
		t.Fatalf("AddProcess() returned environment values: %#v", process.EnvironmentOverrides)
	}

	runtimeState, err := client.Start(ctx, "project", "api")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if runtimeState.State != supervision.ProcessStateRunning {
		t.Fatalf("start state = %q, want running", runtimeState.State)
	}
	stopped, err := client.Stop(ctx, "project", "api")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if stopped.State != supervision.ProcessStateStopped {
		t.Fatalf("stop state = %q, want stopped", stopped.State)
	}
	processes, err := client.ListProcesses(ctx, "project")
	if err != nil {
		t.Fatalf("ListProcesses() error = %v", err)
	}
	if len(processes) != 1 || processes[0].EnvironmentOverrides != nil {
		t.Fatalf("listed processes = %#v, want one redacted definition", processes)
	}
}

func TestServerRejectsIncompatibleVersionBeforeMutation(t *testing.T) {
	client, stopServer := startTestServer(t)
	defer stopServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, err := DialLocal(ctx, client.endpoint)
	if err != nil {
		t.Fatalf("DialLocal() error = %v", err)
	}
	defer connection.Close()
	payload, _ := json.Marshal(projectInput{ID: "project", RootDirectory: t.TempDir()})
	request := Request{Version: ProtocolVersion + 1, ID: "request", Operation: OperationProjectAdd, Payload: payload}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	var response Response
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if response.Error == nil || response.Error.Code != "incompatible_version" {
		t.Fatalf("response error = %#v, want incompatible_version", response.Error)
	}
	projects, err := client.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("projects = %#v, want no mutation", projects)
	}
}

func startTestServer(t *testing.T) (*Client, func()) {
	t.Helper()
	endpoint, err := testEndpoint(t)
	if err != nil {
		t.Fatalf("testEndpoint() error = %v", err)
	}
	listener, err := ListenLocal(endpoint)
	if err != nil {
		t.Fatalf("ListenLocal() error = %v", err)
	}
	registry := supervision.NewRegistry()
	store := supervision.NewFileRegistry(t.TempDir() + string(os.PathSeparator) + "state" + string(os.PathSeparator) + "registry.json")
	manager, err := supervision.NewLifecycleManager(registry, store, 1024, 100)
	if err != nil {
		listener.Close()
		t.Fatalf("NewLifecycleManager() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	go func() { serverDone <- NewServer(manager).Serve(ctx, listener) }()
	return NewClient(endpoint), func() {
		cancel()
		if err := <-serverDone; err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	}
}

func TestControlHelperProcess(t *testing.T) {
	if os.Getenv(controlHelperEnvironment) != "1" {
		return
	}
	_, _ = os.Stdout.WriteString("ready\n")
	for {
		time.Sleep(time.Second)
	}
}
