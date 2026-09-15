//go:build windows

package supervision

import (
	"context"
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

func windowsProcessHasExited(processID uint32) bool {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, processID)
	if err != nil {
		return true
	}
	defer windows.CloseHandle(handle)
	result, err := windows.WaitForSingleObject(handle, 0)
	return err == nil && result == windows.WAIT_OBJECT_0
}
