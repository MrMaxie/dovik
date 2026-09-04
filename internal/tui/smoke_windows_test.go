//go:build windows

package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/supervision"
)

const tuiHelperEnvironment = "DOVIK_TUI_HELPER"

func TestWindowsNamedPipeTUILifecycleSmoke(t *testing.T) {
	endpoint := fmt.Sprintf(`\\.\pipe\dovik-tui-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	listener, err := control.ListenLocal(endpoint)
	if err != nil {
		t.Fatalf("ListenLocal() error = %v", err)
	}
	store := supervision.NewFileRegistry(filepath.Join(t.TempDir(), "state", "registry.json"))
	manager, err := supervision.NewLifecycleManager(supervision.NewRegistry(), store, 4096, 100)
	if err != nil {
		listener.Close()
		t.Fatalf("NewLifecycleManager() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- control.NewServer(manager).Serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	})

	client := control.NewClient(endpoint)
	root := t.TempDir()
	if _, err := client.AddProject(ctx, supervision.ProjectDefinition{ID: "project", RootDirectory: root}); err != nil {
		t.Fatalf("AddProject() error = %v", err)
	}
	if _, err := client.AddProcess(ctx, supervision.ProcessDefinition{
		ProjectID: "project", ID: "worker", Command: os.Args[0],
		Arguments:            []string{"-test.run=TestTUIHelperProcess"},
		EnvironmentOverrides: map[string]string{tuiHelperEnvironment: "1"},
	}); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}

	model := NewModel(ctx, client)
	loaded, initialRefresh := updateModel(t, model, model.loadRegistryCmd()())
	if initialRefresh == nil {
		t.Fatal("registry load did not request initial status")
	}
	loaded, _ = updateModel(t, loaded, initialRefresh())
	started, actionCommand := updateModel(t, loaded, key("s"))
	if actionCommand == nil {
		t.Fatal("start did not create an action command")
	}
	started, refreshCommand := updateModel(t, started, actionCommand())
	if refreshCommand == nil {
		t.Fatal("start did not request a refresh")
	}
	started, _ = updateModel(t, started, refreshCommand())
	if !strings.Contains(started.View().Content, "state=running") {
		t.Fatalf("TUI view does not show running state:\n%s", started.View().Content)
	}

	stopping, stopCommand := updateModel(t, started, key("x"))
	if stopCommand == nil {
		t.Fatal("stop did not create an action command")
	}
	stopped, _ := updateModel(t, stopping, stopCommand())
	if stopped.runtime.State != supervision.ProcessStateStopped {
		t.Fatalf("runtime state = %s, want stopped", stopped.runtime.State)
	}
}

func TestTUIHelperProcess(t *testing.T) {
	if os.Getenv(tuiHelperEnvironment) != "1" {
		return
	}
	_, _ = os.Stdout.WriteString("tui-ready\n")
	for {
		time.Sleep(time.Second)
	}
}
