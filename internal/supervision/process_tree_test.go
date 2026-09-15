package supervision

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

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
