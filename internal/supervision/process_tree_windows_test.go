//go:build windows

package supervision

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsJobObjectStopsDescendantProcess(t *testing.T) {
	manager, _ := testLifecycleManager(t, "spawn-child")
	if _, err := manager.Start("project", "api"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	childPID := waitForChildPID(t, manager)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := manager.Stop(ctx, "project", "api"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if windowsProcessHasExited(childPID) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("descendant process %d remained alive after Job Object stop", childPID)
}

func waitForChildPID(t *testing.T, manager *LifecycleManager) uint32 {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		tail, err := manager.Logs("project", "api", 0, 20)
		if err != nil {
			t.Fatalf("Logs() error = %v", err)
		}
		var output strings.Builder
		for _, event := range tail.Events {
			if event.Stream == OutputStreamStdout {
				output.Write(event.Data)
			}
		}
		line := strings.TrimSpace(output.String())
		if strings.HasPrefix(line, "child=") {
			pid, err := strconv.ParseUint(strings.TrimPrefix(line, "child="), 10, 32)
			if err != nil {
				t.Fatalf("parse child PID from %q: %v", line, err)
			}
			return uint32(pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("child PID was not captured")
	return 0
}

func windowsProcessHasExited(processID uint32) bool {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, processID)
	if err != nil {
		return true
	}
	defer windows.CloseHandle(handle)
	result, err := windows.WaitForSingleObject(handle, 0)
	return err == nil && result == windows.WAIT_OBJECT_0
}
