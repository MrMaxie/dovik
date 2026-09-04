package supervision

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type recordingPersister struct {
	saves int
	err   error
}

func (persister *recordingPersister) Save(*Registry) error {
	persister.saves++
	return persister.err
}

func TestLifecycleManagerRecordsNaturalExitAndOutput(t *testing.T) {
	manager, _ := testLifecycleManager(t, "exit", "0")
	runtimeState, err := manager.Start("project", "api")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if runtimeState.State != ProcessStateRunning {
		t.Fatalf("start state = %q, want %q", runtimeState.State, ProcessStateRunning)
	}

	runtimeState = waitForRuntimeState(t, manager, ProcessStateExited)
	if runtimeState.ExitCode == nil || *runtimeState.ExitCode != 0 {
		t.Fatalf("ExitCode = %v, want 0", runtimeState.ExitCode)
	}
	if runtimeState.TerminationReason != TerminationReasonNaturalExit {
		t.Fatalf("TerminationReason = %q, want %q", runtimeState.TerminationReason, TerminationReasonNaturalExit)
	}

	tail, err := manager.Logs("project", "api", 0, 10)
	if err != nil {
		t.Fatalf("Logs() error = %v", err)
	}
	if len(tail.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(tail.Events))
	}
	streams := map[OutputStream]string{}
	for index, event := range tail.Events {
		if event.Sequence != uint64(index+1) {
			t.Errorf("event sequence = %d, want %d", event.Sequence, index+1)
		}
		streams[event.Stream] = string(event.Data)
	}
	if streams[OutputStreamStdout] != "stdout\n" || streams[OutputStreamStderr] != "stderr\n" {
		t.Errorf("captured streams = %#v, want stdout and stderr", streams)
	}
}

func TestLifecycleManagerPreventsDuplicateActiveRuntime(t *testing.T) {
	manager, _ := testLifecycleManager(t, "wait")
	first, err := manager.Start("project", "api")
	if err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	second, err := manager.Start("project", "api")
	if err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	if second.InstanceID != first.InstanceID || second.PID == nil || first.PID == nil || *second.PID != *first.PID {
		t.Fatalf("second runtime = %#v, want existing runtime %#v", second, first)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stopped, err := manager.Stop(ctx, "project", "api")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if stopped.State != ProcessStateStopped {
		t.Fatalf("stop state = %q, want %q", stopped.State, ProcessStateStopped)
	}
	if stopped.TerminationReason != TerminationReasonRequestedStop {
		t.Fatalf("stop reason = %q, want %q", stopped.TerminationReason, TerminationReasonRequestedStop)
	}
}

func TestLifecycleManagerRestartCreatesNewRuntime(t *testing.T) {
	manager, _ := testLifecycleManager(t, "wait")
	first, err := manager.Start("project", "api")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	second, err := manager.Restart(ctx, "project", "api")
	if err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	if second.InstanceID == first.InstanceID {
		t.Fatalf("restart InstanceID = %q, want a new instance", second.InstanceID)
	}
	if second.State != ProcessStateRunning {
		t.Fatalf("restart state = %q, want %q", second.State, ProcessStateRunning)
	}
	if _, err := manager.Stop(ctx, "project", "api"); err != nil {
		t.Fatalf("final Stop() error = %v", err)
	}
}

func TestLifecycleManagerStopInactiveIsIdempotent(t *testing.T) {
	manager, persister := testLifecycleManager(t, "wait")
	stopped, err := manager.Stop(context.Background(), "project", "api")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if stopped.State != ProcessStateStopped {
		t.Fatalf("State = %q, want %q", stopped.State, ProcessStateStopped)
	}
	if persister.saves != 0 {
		t.Fatalf("persister saves = %d, want 0", persister.saves)
	}
}

func TestLifecycleManagerShutdownStopsOwnedRuntime(t *testing.T) {
	manager, _ := testLifecycleManager(t, "wait")
	if _, err := manager.Start("project", "api"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	runtimeState, exists, err := manager.Status("project", "api")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !exists || runtimeState.State != ProcessStateStopped {
		t.Fatalf("runtime = %#v, exists = %t, want stopped runtime", runtimeState, exists)
	}
}

func TestLifecycleManagerRecordsStartFailure(t *testing.T) {
	manager, _ := testLifecycleManager(t, "wait")
	manager.registry.Projects["project"].Processes["api"].Definition.Command = filepath.Join(t.TempDir(), "missing-command")
	runtimeState, err := manager.Start("project", "api")
	if err == nil {
		t.Fatal("Start() error = nil, want start failure")
	}
	if runtimeState.State != ProcessStateFailed {
		t.Fatalf("State = %q, want %q", runtimeState.State, ProcessStateFailed)
	}
	if runtimeState.TerminationReason != TerminationReasonStartFailed {
		t.Fatalf("TerminationReason = %q, want %q", runtimeState.TerminationReason, TerminationReasonStartFailed)
	}
}

func TestRandomRuntimeInstanceIDUsesUUIDShape(t *testing.T) {
	instanceID, err := randomRuntimeInstanceID()
	if err != nil {
		t.Fatalf("randomRuntimeInstanceID() error = %v", err)
	}
	parts := strings.Split(string(instanceID), "-")
	if len(parts) != 5 || len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		t.Fatalf("instance ID = %q, want UUID shape", instanceID)
	}
}

func testLifecycleManager(t *testing.T, helperArguments ...string) (*LifecycleManager, *recordingPersister) {
	t.Helper()
	arguments := []string{"-test.run=TestProcessHelperProcess", "--"}
	arguments = append(arguments, helperArguments...)
	registry := NewRegistry()
	registry.Projects["project"] = &RegisteredProject{
		Definition: ProjectDefinition{ID: "project", RootDirectory: t.TempDir()},
		Processes: map[ProcessID]*RegisteredProcess{
			"api": {
				Definition: ProcessDefinition{
					ProjectID:            "project",
					ID:                   "api",
					Command:              os.Args[0],
					Arguments:            arguments,
					EnvironmentOverrides: map[string]string{processHelperEnvironment: "1"},
				},
			},
		},
	}
	persister := &recordingPersister{}
	manager, err := NewLifecycleManager(registry, persister, 1024, 100)
	if err != nil {
		t.Fatalf("NewLifecycleManager() error = %v", err)
	}
	return manager, persister
}

func waitForRuntimeState(t *testing.T, manager *LifecycleManager, wanted ProcessState) ProcessRuntime {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runtimeState, exists, err := manager.Status("project", "api")
		if err != nil {
			t.Fatalf("Status() error = %v", err)
		}
		if exists && runtimeState.State == wanted {
			return runtimeState
		}
		time.Sleep(10 * time.Millisecond)
	}
	runtimeState, _, _ := manager.Status("project", "api")
	t.Fatalf("runtime state = %q, want %q", runtimeState.State, wanted)
	return ProcessRuntime{}
}
