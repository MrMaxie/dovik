//go:build !windows

package supervision

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

func TestUnixProcessGroupStopsDescendantProcess(t *testing.T) {
	manager, _ := testLifecycleManager(t, "spawn-child")
	if _, err := manager.Start("project", "api"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	childPID := int(waitForChildPID(t, manager))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := manager.Stop(ctx, "project", "api"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(childPID, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("descendant process %d remained alive after process group stop", childPID)
}
