package supervision

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryOperationsValidateAndPersistDefinitions(t *testing.T) {
	registry := NewRegistry()
	persister := &recordingPersister{}
	manager, err := NewLifecycleManager(registry, persister, 1024, 100)
	if err != nil {
		t.Fatalf("NewLifecycleManager() error = %v", err)
	}
	root := t.TempDir()
	project, err := manager.AddProject(ProjectDefinition{ID: "project", RootDirectory: root})
	if err != nil {
		t.Fatalf("AddProject() error = %v", err)
	}
	if project.ID != "project" {
		t.Fatalf("project = %#v", project)
	}

	process, err := manager.AddProcess(ProcessDefinition{
		ProjectID:            "project",
		ID:                   "api",
		Command:              os.Args[0],
		Arguments:            []string{"--version"},
		EnvironmentOverrides: map[string]string{"SECRET": "value"},
	})
	if err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	if process.EnvironmentOverrides != nil {
		t.Fatalf("returned environment = %#v, want omitted values", process.EnvironmentOverrides)
	}
	processes, err := manager.ListProcesses("project")
	if err != nil {
		t.Fatalf("ListProcesses() error = %v", err)
	}
	if len(processes) != 1 || processes[0].EnvironmentOverrides != nil {
		t.Fatalf("listed processes = %#v, want one redacted definition", processes)
	}
	if persister.saves != 2 {
		t.Fatalf("persister saves = %d, want 2", persister.saves)
	}
}

func TestRegistryOperationsRejectUnsafeRemovalAndInvalidRoots(t *testing.T) {
	manager, _ := testLifecycleManager(t, "wait")
	if _, err := manager.AddProject(ProjectDefinition{ID: "relative", RootDirectory: "relative"}); err == nil {
		t.Fatal("AddProject() error = nil, want relative root error")
	}
	if err := manager.RemoveProject("project"); err == nil {
		t.Fatal("RemoveProject() error = nil, want owned process error")
	}
	if _, err := manager.Start("project", "api"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := manager.RemoveProcess("project", "api"); err == nil {
		t.Fatal("RemoveProcess() error = nil, want active process error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := manager.Stop(ctx, "project", "api"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := manager.RemoveProcess("project", "api"); err != nil {
		t.Fatalf("RemoveProcess() error = %v", err)
	}
	if err := manager.RemoveProject("project"); err != nil {
		t.Fatalf("RemoveProject() error = %v", err)
	}
}

func TestAddProcessValidatesRelativeWorkingDirectory(t *testing.T) {
	manager, _ := testLifecycleManager(t, "wait")
	if _, err := manager.AddProcess(ProcessDefinition{
		ProjectID:        "project",
		ID:               "worker",
		Command:          os.Args[0],
		WorkingDirectory: filepath.Join("missing", "directory"),
	}); err == nil {
		t.Fatal("AddProcess() error = nil, want missing working directory error")
	}
}
