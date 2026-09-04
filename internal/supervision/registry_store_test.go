package supervision

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFileRegistrySaveLoadAndReplace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", registryFileName)
	store := NewFileRegistry(path)
	registry := testRegistry(t, ProcessStateRunning)

	if err := store.Save(registry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	registry.Projects["project"].Processes["api"].Definition.Arguments = []string{"--port", "7882"}
	if err := store.Save(registry); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	arguments := loaded.Projects["project"].Processes["api"].Definition.Arguments
	if len(arguments) != 2 || arguments[1] != "7882" {
		t.Fatalf("loaded arguments = %v, want updated replacement", arguments)
	}

	temporaryFiles, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".registry-*.tmp"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary registry files remain: %v", temporaryFiles)
	}
}

func TestFileRegistryLoadMissingReturnsEmptyRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", registryFileName)
	registry, err := NewFileRegistry(path).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(registry.Projects) != 0 {
		t.Fatalf("len(Projects) = %d, want 0", len(registry.Projects))
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat() error = %v, want missing registry file", err)
	}
}

func TestFileRegistryRejectsRelativePath(t *testing.T) {
	store := NewFileRegistry(registryFileName)
	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want relative path error")
	}
	if err := store.Save(NewRegistry()); err == nil {
		t.Fatal("Save() error = nil, want relative path error")
	}
}

func TestFileRegistryRejectsCorruptContentWithoutReplacingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), registryFileName)
	content := []byte(`{"version":1,"projects":[`) // Deliberately incomplete.
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := NewFileRegistry(path).Load(); err == nil {
		t.Fatal("Load() error = nil, want corrupt registry error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("corrupt registry was changed to %q", got)
	}
}

func TestFileRegistryLoadAndReconcilePersistsFailedRuntime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", registryFileName)
	store := NewFileRegistry(path)
	registry := testRegistry(t, ProcessStateRunning)
	if err := store.Save(registry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reconciledAt := time.Date(2026, time.September, 1, 14, 0, 0, 0, time.UTC)
	loaded, err := store.LoadAndReconcile(reconciledAt)
	if err != nil {
		t.Fatalf("LoadAndReconcile() error = %v", err)
	}
	runtimeState := loaded.Projects["project"].Processes["api"].Runtime
	if runtimeState.State != ProcessStateFailed {
		t.Fatalf("State = %q, want %q", runtimeState.State, ProcessStateFailed)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() after reconciliation error = %v", err)
	}
	persisted := reloaded.Projects["project"].Processes["api"].Runtime
	if persisted.State != ProcessStateFailed || persisted.TerminationReason != DaemonRestartTerminationReason {
		t.Fatalf("persisted runtime = %#v, want failed daemon-restart state", persisted)
	}
	if persisted.FinishedAt == nil || !persisted.FinishedAt.Equal(reconciledAt) {
		t.Fatalf("persisted FinishedAt = %v, want %v", persisted.FinishedAt, reconciledAt)
	}
}

func TestFileRegistryCreatesPrivateDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", registryFileName)
	if err := NewFileRegistry(path).Save(NewRegistry()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Windows permission bits do not represent the directory ACL")
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if permissions := info.Mode().Perm(); permissions&0o077 != 0 {
		t.Fatalf("directory permissions = %04o, want private", permissions)
	}
}

func TestDefaultRegistryPathIsAbsolute(t *testing.T) {
	path, err := DefaultRegistryPath()
	if err != nil {
		t.Fatalf("DefaultRegistryPath() error = %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("DefaultRegistryPath() = %q, want absolute path", path)
	}
	if !strings.EqualFold(filepath.Base(path), registryFileName) {
		t.Fatalf("DefaultRegistryPath() base = %q, want %q", filepath.Base(path), registryFileName)
	}
}

func testRegistry(t *testing.T, state ProcessState) *Registry {
	t.Helper()
	root := t.TempDir()
	pid := 42
	startedAt := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	registry := NewRegistry()
	registry.Projects["project"] = &RegisteredProject{
		Definition: ProjectDefinition{ID: "project", RootDirectory: root},
		Processes: map[ProcessID]*RegisteredProcess{
			"api": {
				Definition: ProcessDefinition{
					ProjectID:            "project",
					ID:                   "api",
					Command:              "server",
					Arguments:            []string{"--port", "7881"},
					EnvironmentOverrides: map[string]string{"TOKEN": "private"},
				},
				Runtime: &ProcessRuntime{
					ProjectID:  "project",
					ProcessID:  "api",
					InstanceID: "runtime-1",
					PID:        &pid,
					State:      state,
					StartedAt:  &startedAt,
				},
			},
		},
	}
	return registry
}
