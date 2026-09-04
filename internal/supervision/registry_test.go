package supervision

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRegistryDocumentRoundTrip(t *testing.T) {
	startedAt := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	pid := 42
	registry := NewRegistry()
	registry.Projects["project"] = &RegisteredProject{
		Definition: ProjectDefinition{ID: "project", RootDirectory: t.TempDir()},
		Processes: map[ProcessID]*RegisteredProcess{
			"api": {
				Definition: ProcessDefinition{
					ProjectID:            "project",
					ID:                   "api",
					Command:              "server",
					Arguments:            []string{"--port", "7881"},
					WorkingDirectory:     filepath.Join("work", "api"),
					EnvironmentOverrides: map[string]string{"TOKEN": "private"},
				},
				Runtime: &ProcessRuntime{
					ProjectID:  "project",
					ProcessID:  "api",
					InstanceID: "runtime-1",
					PID:        &pid,
					State:      ProcessStateRunning,
					StartedAt:  &startedAt,
				},
			},
		},
	}

	document, err := registryToDocument(registry)
	if err != nil {
		t.Fatalf("registryToDocument() error = %v", err)
	}
	got, err := documentToRegistry(document)
	if err != nil {
		t.Fatalf("documentToRegistry() error = %v", err)
	}
	if !reflect.DeepEqual(got, registry) {
		t.Fatalf("round trip registry = %#v, want %#v", got, registry)
	}
}

func TestRegistryReconcileAfterDaemonRestart(t *testing.T) {
	reconciledAt := time.Date(2026, time.September, 1, 13, 0, 0, 0, time.UTC)
	registry := NewRegistry()
	project := &RegisteredProject{
		Definition: ProjectDefinition{ID: "project", RootDirectory: t.TempDir()},
		Processes:  make(map[ProcessID]*RegisteredProcess),
	}
	registry.Projects["project"] = project

	states := []ProcessState{
		ProcessStateStopped,
		ProcessStateStarting,
		ProcessStateRunning,
		ProcessStateStopping,
		ProcessStateExited,
		ProcessStateFailed,
	}
	for _, state := range states {
		processID := ProcessID(state)
		project.Processes[processID] = &RegisteredProcess{
			Definition: ProcessDefinition{ProjectID: "project", ID: processID, Command: "server"},
			Runtime: &ProcessRuntime{
				ProjectID:  "project",
				ProcessID:  processID,
				InstanceID: RuntimeInstanceID("runtime-" + state),
				State:      state,
			},
		}
	}

	changed, err := registry.ReconcileAfterDaemonRestart(reconciledAt)
	if err != nil {
		t.Fatalf("ReconcileAfterDaemonRestart() error = %v", err)
	}
	if !changed {
		t.Fatal("ReconcileAfterDaemonRestart() changed = false, want true")
	}

	for _, state := range states {
		runtime := project.Processes[ProcessID(state)].Runtime
		if state.IsActive() {
			if runtime.State != ProcessStateFailed {
				t.Errorf("runtime from %q state = %q, want %q", state, runtime.State, ProcessStateFailed)
			}
			if runtime.FinishedAt == nil || !runtime.FinishedAt.Equal(reconciledAt) {
				t.Errorf("runtime from %q FinishedAt = %v, want %v", state, runtime.FinishedAt, reconciledAt)
			}
			if runtime.TerminationReason != DaemonRestartTerminationReason {
				t.Errorf("runtime from %q reason = %q, want %q", state, runtime.TerminationReason, DaemonRestartTerminationReason)
			}
			continue
		}
		if runtime.State != state {
			t.Errorf("terminal runtime from %q state = %q, want unchanged", state, runtime.State)
		}
	}
}

func TestDocumentToRegistryRejectsDuplicateProcesses(t *testing.T) {
	document := registryDocument{
		Version: registryDocumentVersion,
		Projects: []projectDocument{
			{
				ID:            "project",
				RootDirectory: t.TempDir(),
				Processes: []processDocument{
					{ID: "api", Command: "server"},
					{ID: "api", Command: "server"},
				},
			},
		},
	}

	if _, err := documentToRegistry(document); err == nil {
		t.Fatal("documentToRegistry() error = nil, want duplicate process error")
	}
}
