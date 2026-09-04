package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MrMaxie/dovik/internal/supervision"
)

type fakeClient struct {
	projects  []supervision.ProjectDefinition
	processes map[supervision.ProjectID][]supervision.ProcessDefinition
	runtimes  map[processKey]supervision.ProcessRuntime
	tails     map[processKey]supervision.OutputTail
	listErr   error
	actionErr error
	starts    int
	stops     int
	restarts  int
}

func (client *fakeClient) AddProject(context.Context, supervision.ProjectDefinition) (supervision.ProjectDefinition, error) {
	return supervision.ProjectDefinition{}, errors.New("not used")
}
func (client *fakeClient) RemoveProject(context.Context, supervision.ProjectID) error {
	return errors.New("not used")
}
func (client *fakeClient) ListProjects(context.Context) ([]supervision.ProjectDefinition, error) {
	return append([]supervision.ProjectDefinition(nil), client.projects...), client.listErr
}
func (client *fakeClient) AddProcess(context.Context, supervision.ProcessDefinition) (supervision.ProcessDefinition, error) {
	return supervision.ProcessDefinition{}, errors.New("not used")
}
func (client *fakeClient) RemoveProcess(context.Context, supervision.ProjectID, supervision.ProcessID) error {
	return errors.New("not used")
}
func (client *fakeClient) ListProcesses(_ context.Context, projectID supervision.ProjectID) ([]supervision.ProcessDefinition, error) {
	return append([]supervision.ProcessDefinition(nil), client.processes[projectID]...), client.listErr
}
func (client *fakeClient) Start(_ context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, error) {
	client.starts++
	return client.runtimes[processKey{projectID: projectID, processID: processID}], client.actionErr
}
func (client *fakeClient) Stop(_ context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, error) {
	client.stops++
	return client.runtimes[processKey{projectID: projectID, processID: processID}], client.actionErr
}
func (client *fakeClient) Restart(_ context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, error) {
	client.restarts++
	return client.runtimes[processKey{projectID: projectID, processID: processID}], client.actionErr
}
func (client *fakeClient) Status(_ context.Context, projectID supervision.ProjectID, processID supervision.ProcessID) (supervision.ProcessRuntime, bool, error) {
	runtimeState, exists := client.runtimes[processKey{projectID: projectID, processID: processID}]
	return runtimeState, exists, client.listErr
}
func (client *fakeClient) Logs(_ context.Context, projectID supervision.ProjectID, processID supervision.ProcessID, _ uint64, _ int) (supervision.OutputTail, error) {
	return client.tails[processKey{projectID: projectID, processID: processID}], client.listErr
}

func TestModelLoadsStatusAndAttributedOutput(t *testing.T) {
	key := processKey{projectID: "project", processID: "api"}
	client := &fakeClient{
		projects:  []supervision.ProjectDefinition{{ID: key.projectID, RootDirectory: `C:\project`}},
		processes: map[supervision.ProjectID][]supervision.ProcessDefinition{key.projectID: {{ProjectID: key.projectID, ID: key.processID, Command: "server"}}},
		runtimes:  map[processKey]supervision.ProcessRuntime{key: {ProjectID: key.projectID, ProcessID: key.processID, InstanceID: "instance", State: supervision.ProcessStateRunning}},
		tails: map[processKey]supervision.OutputTail{key: {
			Truncated: true,
			Events: []supervision.OutputEvent{
				{Sequence: 2, Stream: supervision.OutputStreamStderr, Data: []byte("warning\n")},
				{Sequence: 1, Stream: supervision.OutputStreamStdout, Data: []byte("ready\n")},
			},
		}},
	}
	model := NewModel(context.Background(), client)
	model.width, model.height = 100, 30

	loaded, command := updateModel(t, model, model.loadRegistryCmd()())
	if command == nil {
		t.Fatal("registry load did not request selected process refresh")
	}
	refreshed, _ := updateModel(t, loaded, command())
	view := refreshed.View().Content
	for _, expected := range []string{"project/api", "state=running", "[stdout] ready", "[stderr] warning", "output truncated"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("view missing %q:\n%s", expected, view)
		}
	}
	if strings.Index(view, "[stdout] ready") > strings.Index(view, "[stderr] warning") {
		t.Fatalf("output is not in sequence order:\n%s", view)
	}
}

func TestModelIgnoresStaleRefreshAfterSelectionChange(t *testing.T) {
	client := &fakeClient{
		projects: []supervision.ProjectDefinition{{ID: "project"}},
		processes: map[supervision.ProjectID][]supervision.ProcessDefinition{"project": {
			{ProjectID: "project", ID: "api", Command: "api"},
			{ProjectID: "project", ID: "web", Command: "web"},
		}},
		runtimes: map[processKey]supervision.ProcessRuntime{},
		tails:    map[processKey]supervision.OutputTail{},
	}
	model := NewModel(context.Background(), client)
	loaded, staleCommand := updateModel(t, model, model.loadRegistryCmd()())
	changed, currentCommand := updateModel(t, loaded, key("down"))
	if currentCommand == nil {
		t.Fatal("selection change did not request refresh")
	}
	staleApplied, _ := updateModel(t, changed, staleCommand())
	if staleApplied.selectedKey().processID != "web" || staleApplied.refreshing == false {
		t.Fatalf("stale response changed selection or refresh state: %#v", staleApplied)
	}
	current, _ := updateModel(t, staleApplied, currentCommand())
	if current.selectedKey().processID != "web" || current.refreshing {
		t.Fatalf("current response was not applied: %#v", current)
	}
}

func TestModelSerializesActionsAndKeepsDiagnosticsOnDemand(t *testing.T) {
	process := processKey{projectID: "project", processID: "api"}
	client := &fakeClient{
		projects:  []supervision.ProjectDefinition{{ID: process.projectID}},
		processes: map[supervision.ProjectID][]supervision.ProcessDefinition{process.projectID: {{ProjectID: process.projectID, ID: process.processID}}},
		runtimes:  map[processKey]supervision.ProcessRuntime{process: {ProjectID: process.projectID, ProcessID: process.processID, State: supervision.ProcessStateStopped}},
		tails:     map[processKey]supervision.OutputTail{},
		actionErr: errors.New("raw transport detail"),
	}
	model := NewModel(context.Background(), client)
	loaded, refreshCommand := updateModel(t, model, model.loadRegistryCmd()())
	loaded, _ = updateModel(t, loaded, refreshCommand())
	pending, actionCommand := updateModel(t, loaded, key("s"))
	stillPending, duplicateCommand := updateModel(t, pending, key("s"))
	if actionCommand == nil || duplicateCommand != nil || !stillPending.pending {
		t.Fatal("model did not serialize lifecycle actions")
	}
	failed, _ := updateModel(t, stillPending, actionCommand())
	if client.starts != 1 || strings.Contains(failed.View().Content, "raw transport detail") {
		t.Fatalf("primary view exposed diagnostics or action count = %d:\n%s", client.starts, failed.View().Content)
	}
	detailed, _ := updateModel(t, failed, key("d"))
	if !strings.Contains(detailed.View().Content, "Diagnostic: raw transport detail") {
		t.Fatalf("details did not expose diagnostic on demand:\n%s", detailed.View().Content)
	}
}

func TestModelHandlesMinimumTerminalSize(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	updated, _ := updateModel(t, model, tea.WindowSizeMsg{Width: 40, Height: 10})
	view := updated.View().Content
	if !strings.Contains(view, "Terminal too small") || !strings.Contains(view, "? help  q quit") {
		t.Fatalf("small terminal view = %q", view)
	}
}

func TestModelResetsSequenceWhenRuntimeChanges(t *testing.T) {
	process := processKey{projectID: "project", processID: "api"}
	client := &fakeClient{
		runtimes: map[processKey]supervision.ProcessRuntime{process: {
			ProjectID: "project", ProcessID: "api", InstanceID: "new", State: supervision.ProcessStateRunning,
		}},
		tails: map[processKey]supervision.OutputTail{process: {
			Events: []supervision.OutputEvent{{Sequence: 1, Stream: supervision.OutputStreamStdout, Data: []byte("new runtime")}},
		}},
	}
	model := NewModel(context.Background(), client)
	model.items = []processItem{{key: process}}
	model.hasRuntime = true
	model.runtime = supervision.ProcessRuntime{ProjectID: "project", ProcessID: "api", InstanceID: "old", State: supervision.ProcessStateRunning}
	model.events = []supervision.OutputEvent{{Sequence: 20, Data: []byte("old runtime")}}
	model.lastSeq = 20
	model.refreshing = true

	updated, _ := updateModel(t, model, model.refreshCmd()())
	if updated.lastSeq != 1 || len(updated.events) != 1 || string(updated.events[0].Data) != "new runtime" {
		t.Fatalf("runtime output was not reset: lastSeq=%d events=%#v", updated.lastSeq, updated.events)
	}
}

func updateModel(t *testing.T, model Model, message tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, command := model.Update(message)
	result, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model type = %T", updated)
	}
	return result, command
}

func key(value string) tea.KeyPressMsg {
	if value == "down" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(value[0]), Text: value})
}

func TestFormatTime(t *testing.T) {
	value := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	if got := formatTime(&value); got != "2026-09-04T12:00:00Z" {
		t.Fatalf("formatTime() = %q", got)
	}
}
