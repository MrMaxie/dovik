package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

type fakeDaemonLauncher struct {
	starts int
	start  func() (int, error)
}

func (launcher *fakeDaemonLauncher) Start() (int, error) {
	launcher.starts++
	return launcher.start()
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
	model.presentation = newPresentation(false)
	model.width, model.height = 100, 30

	loaded, command := updateModel(t, model, model.loadRegistryCmd()())
	if command == nil {
		t.Fatal("registry load did not request selected process refresh")
	}
	refreshed, _ := updateModel(t, loaded, command())
	view := refreshed.View().Content
	for _, expected := range []string{"project / api", "[RUNNING]", "Command  server", "[stdout] ready", "[stderr] warning", "Older output is unavailable"} {
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
	model.presentation = newPresentation(false)
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
	model.presentation = newPresentation(false)
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
	if !strings.Contains(detailed.View().Content, "Diagnostic  raw transport detail") {
		t.Fatalf("details did not expose diagnostic on demand:\n%s", detailed.View().Content)
	}
	refreshed, _ := updateModel(t, failed, refreshMsg{
		key:        processKey{projectID: "project", processID: "api"},
		generation: failed.generation,
		runtime:    supervision.ProcessRuntime{ProjectID: "project", ProcessID: "api", State: supervision.ProcessStateStopped},
	})
	if !strings.Contains(refreshed.View().Content, "Start did not complete. Press s to retry.") {
		t.Fatalf("successful polling hid the action failure before a retry:\n%s", refreshed.View().Content)
	}
}

func TestModelStartsUnavailableDaemonAndWaitsForProtocolReadiness(t *testing.T) {
	client := &fakeClient{listErr: errors.New("daemon unavailable")}
	launcher := &fakeDaemonLauncher{start: func() (int, error) {
		client.listErr = nil
		return 4242, nil
	}}
	model := NewModelWithDaemonLauncher(context.Background(), client, launcher)
	model.presentation = newPresentation(false)
	model.width, model.height = 100, 30
	unavailable, _ := updateModel(t, model, model.loadRegistryCmd()())
	if view := unavailable.View().Content; !strings.Contains(view, "Press s to start the daemon") || !strings.Contains(view, "[s] Start daemon") {
		t.Fatalf("unavailable view omitted daemon start action:\n%s", view)
	}
	pending, command := updateModel(t, unavailable, key("s"))
	stillPending, duplicate := updateModel(t, pending, key("s"))
	if command == nil || duplicate != nil || !stillPending.daemonStarting || launcher.starts != 0 {
		t.Fatalf("daemon launch was not serialized: pending=%v starts=%d", stillPending.daemonStarting, launcher.starts)
	}
	ready, _ := updateModel(t, stillPending, command())
	if launcher.starts != 1 || ready.daemonStarting || ready.registry != registryEmpty {
		t.Fatalf("daemon did not become ready: starts=%d state=%v starting=%v", launcher.starts, ready.registry, ready.daemonStarting)
	}
}

func TestModelAcceptsCompatibleDaemonAfterLosingLaunchRace(t *testing.T) {
	client := &fakeClient{listErr: errors.New("daemon unavailable")}
	launcher := &fakeDaemonLauncher{start: func() (int, error) {
		client.listErr = nil
		return 0, errors.New("endpoint already claimed")
	}}
	model := NewModelWithDaemonLauncher(context.Background(), client, launcher)
	model.loading = false
	model.registry = registryUnavailable
	pending, command := updateModel(t, model, key("s"))
	ready, _ := updateModel(t, pending, command())
	if ready.registry != registryEmpty || ready.diagnostic != "" || ready.notice != "" {
		t.Fatalf("compatible concurrent daemon was reported as failure: %#v", ready)
	}
}

func TestModelKeepsDaemonLaunchFailureDetailsOnDemand(t *testing.T) {
	client := &fakeClient{listErr: errors.New("transport detail")}
	launcher := &fakeDaemonLauncher{start: func() (int, error) { return 0, errors.New("process detail") }}
	model := NewModelWithDaemonLauncher(context.Background(), client, launcher)
	model.presentation = newPresentation(false)
	model.width, model.height = 100, 30
	model.loading = false
	model.registry = registryUnavailable
	model.daemonReadyTimeout = 10 * time.Millisecond
	model.daemonRetryDelay = time.Millisecond
	pending, command := updateModel(t, model, key("s"))
	failed, _ := updateModel(t, pending, command())
	view := failed.View().Content
	if !strings.Contains(view, "Daemon did not start or become available") || strings.Contains(view, "process detail") || strings.Contains(view, "transport detail") {
		t.Fatalf("primary daemon failure was not concise:\n%s", view)
	}
	detailed, _ := updateModel(t, failed, key("d"))
	if detailView := detailed.View().Content; !strings.Contains(detailView, "process detail") || !strings.Contains(detailView, "transport detail") {
		t.Fatalf("daemon diagnostic details were not available on demand:\n%s", detailView)
	}
}

func TestModelKeepsConnectionOnlyUnavailableStateWithoutLauncher(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{listErr: errors.New("daemon unavailable")})
	model.presentation = newPresentation(false)
	model.width, model.height = 100, 30
	unavailable, _ := updateModel(t, model, model.loadRegistryCmd()())
	updated, command := updateModel(t, unavailable, key("s"))
	if command != nil || updated.daemonStarting || strings.Contains(updated.View().Content, "Start daemon") {
		t.Fatalf("connection-only TUI offered daemon launch:\n%s", updated.View().Content)
	}
}

func TestModelHandlesMinimumTerminalSize(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	model.presentation = newPresentation(false)
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
	model.presentation = newPresentation(false)
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

func TestPresentationUsesWideAndCompactWorkspaces(t *testing.T) {
	model := Model{
		items:        []processItem{{key: processKey{projectID: "dovik", processID: "api"}, command: "go run ./cmd/api --dev"}},
		statusKnown:  true,
		hasRuntime:   true,
		runtime:      supervision.ProcessRuntime{State: supervision.ProcessStateRunning},
		presentation: newPresentation(false),
	}

	model.width, model.height = 120, 34
	wide := model.View().Content
	wideHeader := strings.Split(wide, "\n")[1]
	if lipgloss.Width(wide) != 120 || lipgloss.Height(wide) != 34 || strings.Count(wideHeader, "│") != 1 || !strings.Contains(wide, "dovik / api") {
		t.Fatalf("wide workspace dimensions = %dx%d:\n%s", lipgloss.Width(wide), lipgloss.Height(wide), wide)
	}

	model.width, model.height = 80, 24
	compact := model.View().Content
	compactLines := strings.Split(compact, "\n")
	if lipgloss.Width(compact) != 80 || lipgloss.Height(compact) != 24 || strings.Contains(compactLines[1], "┌") || !strings.Contains(compact, "Command  go run ./cmd/api --dev") {
		t.Fatalf("compact workspace dimensions = %dx%d:\n%s", lipgloss.Width(compact), lipgloss.Height(compact), compact)
	}
}

func TestPresentationKeepsMeaningWithoutColor(t *testing.T) {
	model := Model{
		width:        100,
		height:       30,
		items:        []processItem{{key: processKey{projectID: "project", processID: "api"}, command: "server"}},
		statusKnown:  true,
		hasRuntime:   true,
		runtime:      supervision.ProcessRuntime{State: supervision.ProcessStateFailed},
		presentation: newPresentation(false),
		notice:       "Restart did not complete. Press r to retry.",
		diagnostic:   "transport detail",
		truncated:    true,
		events: []supervision.OutputEvent{
			{Sequence: 1, Stream: supervision.OutputStreamStdout, Data: []byte("ready\n")},
			{Sequence: 2, Stream: supervision.OutputStreamStderr, Data: []byte("failed\n")},
		},
	}

	view := model.View().Content
	if strings.Contains(view, "\x1b[") {
		t.Fatalf("color-disabled presentation emitted ANSI styling: %q", view)
	}
	for _, expected := range []string{"> project / api", "[FAILED]", "Restart did not complete", "[stdout] ready", "[stderr] failed", "Older output is unavailable"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("color-disabled view missing %q:\n%s", expected, view)
		}
	}
}

func TestPresentationStylesLifecycleStatesSemantically(t *testing.T) {
	p := newPresentation(true)
	running := p.renderState(string(supervision.ProcessStateRunning))
	failed := p.renderState(string(supervision.ProcessStateFailed))
	pending := p.renderState(string(supervision.ProcessStateStarting))
	if running == failed || failed == pending || !strings.Contains(running, "RUNNING") || !strings.Contains(failed, "FAILED") || !strings.Contains(pending, "STARTING") {
		t.Fatalf("state styles are not semantically distinct: running=%q failed=%q pending=%q", running, failed, pending)
	}
}

func TestPresentationKeepsRuntimeDiagnosticsBehindDetails(t *testing.T) {
	pid := 4242
	model := Model{
		width:        80,
		height:       24,
		items:        []processItem{{key: processKey{projectID: "project", processID: "api"}, command: "server"}},
		statusKnown:  true,
		hasRuntime:   true,
		runtime:      supervision.ProcessRuntime{InstanceID: "instance", PID: &pid, State: supervision.ProcessStateRunning},
		presentation: newPresentation(false),
	}
	if view := model.View().Content; strings.Contains(view, "PID") || strings.Contains(view, "instance") {
		t.Fatalf("default operator workspace exposed diagnostics:\n%s", view)
	}
	model.showDetail = true
	view := model.View().Content
	if !strings.Contains(view, "Runtime details") || !strings.Contains(view, "PID  4242") || !strings.Contains(view, "Instance  instance") {
		t.Fatalf("details view omitted runtime diagnostics:\n%s", view)
	}
}

func TestPresentationAttributesEveryOutputLine(t *testing.T) {
	model := Model{
		width:        80,
		height:       24,
		items:        []processItem{{key: processKey{projectID: "project", processID: "api"}}},
		statusKnown:  true,
		presentation: newPresentation(false),
		events: []supervision.OutputEvent{{
			Sequence: 1,
			Stream:   supervision.OutputStreamStdout,
			Data:     []byte("first\nsecond\n"),
		}},
	}
	view := model.View().Content
	if strings.Count(view, "[stdout]") != 2 || !strings.Contains(view, "[stdout] second") {
		t.Fatalf("multiline output lost stream attribution:\n%s", view)
	}
}

func TestResizePreservesOperatorState(t *testing.T) {
	model := Model{
		width:        120,
		height:       34,
		items:        []processItem{{key: processKey{projectID: "project", processID: "api"}}},
		statusKnown:  true,
		showDetail:   true,
		showHelp:     true,
		scroll:       3,
		presentation: newPresentation(false),
	}
	updated, _ := updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})
	if updated.selected != model.selected || updated.scroll != model.scroll || !updated.showDetail || !updated.showHelp {
		t.Fatalf("resize changed operator state: before=%#v after=%#v", model, updated)
	}
}

func TestFormatCommandQuotesArgumentsWithSpaces(t *testing.T) {
	if got := formatCommand("server", []string{"--name", "Dovik API"}); got != `server --name "Dovik API"` {
		t.Fatalf("formatCommand() = %q", got)
	}
}

func TestPresentationShortcutBarReflectsOpenPanel(t *testing.T) {
	model := Model{presentation: newPresentation(false)}
	model.showDetail = true
	if got := model.presentation.renderShortcutBar(model, 80); !strings.Contains(got, "[d] Close details") {
		t.Fatalf("details shortcut did not describe the active action: %q", got)
	}
	model.showHelp = true
	if got := model.presentation.renderShortcutBar(model, 80); !strings.Contains(got, "[?] Close help") {
		t.Fatalf("help shortcut did not describe the active action: %q", got)
	}
}
