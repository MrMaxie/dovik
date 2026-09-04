//go:build integration

package integration

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestLinuxContainerLifecycle(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repositoryRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{Context: repositoryRoot, Dockerfile: "Dockerfile"},
			WaitingFor:     wait.ForLog("Dovik daemon is running.").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start Dovik container: %v", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate Dovik container: %v", err)
		}
	}()

	inspection, err := container.Inspect(ctx)
	if err != nil {
		t.Fatalf("inspect Dovik container: %v", err)
	}
	if len(inspection.NetworkSettings.Ports) != 0 {
		t.Fatalf("published or exposed ports = %v, want none", inspection.NetworkSettings.Ports)
	}

	execDovik(t, ctx, container, "project", "add", "--id", "fixture", "--root", "/tmp")
	execDovik(t, ctx, container,
		"process", "add",
		"--project", "fixture",
		"--id", "worker",
		"--command", "/bin/sh",
		"--arg=-c",
		"--arg=printf 'stdout-line\\n'; printf 'stderr-line\\n' >&2; sleep 300",
	)
	startOutput := execDovik(t, ctx, container, "process", "start", "--project", "fixture", "--process", "worker")
	if !strings.Contains(startOutput, "\trunning\t") {
		t.Fatalf("start output = %q, want running", startOutput)
	}

	deadline := time.Now().Add(10 * time.Second)
	var logs string
	for time.Now().Before(deadline) {
		logs = execDovik(t, ctx, container, "process", "logs", "--project", "fixture", "--process", "worker", "--tail", "20")
		if strings.Contains(logs, "stdout-line") && strings.Contains(logs, "stderr-line") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(logs, "[stdout]") || !strings.Contains(logs, "[stderr]") {
		t.Fatalf("logs output = %q, want attributed stdout and stderr", logs)
	}
	execDovikTUI(t, ctx, container)
	statusAfterTUI := execDovik(t, ctx, container, "process", "status", "--project", "fixture", "--process", "worker")
	if !strings.Contains(statusAfterTUI, "\tstopped\t") {
		t.Fatalf("status after TUI stop = %q, want stopped", statusAfterTUI)
	}
	execDovik(t, ctx, container, "process", "start", "--project", "fixture", "--process", "worker")

	stopOutput := execDovik(t, ctx, container, "process", "stop", "--project", "fixture", "--process", "worker")
	if !strings.Contains(stopOutput, "\tstopped\t") {
		t.Fatalf("stop output = %q, want stopped", stopOutput)
	}
	restartOutput := execDovik(t, ctx, container, "process", "restart", "--project", "fixture", "--process", "worker")
	if !strings.Contains(restartOutput, "\trunning\t") {
		t.Fatalf("restart output = %q, want running", restartOutput)
	}
	execDovik(t, ctx, container, "process", "stop", "--project", "fixture", "--process", "worker")

	stopTimeout := 10 * time.Second
	if err := container.Stop(ctx, &stopTimeout); err != nil {
		t.Fatalf("stop Dovik container: %v", err)
	}
	if err := container.Start(ctx); err != nil {
		t.Fatalf("restart Dovik container: %v", err)
	}
	projects := execDovik(t, ctx, container, "project", "list")
	if !strings.Contains(projects, "fixture\t/tmp") {
		t.Fatalf("projects after daemon restart = %q, want persisted fixture", projects)
	}
}

func execDovikTUI(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()
	command := []string{"sh", "-c", "(sleep 2; printf x; sleep 2; printf q) | script -qec 'dovik tui' /dev/null"}
	exitCode, reader, err := container.Exec(ctx, command, tcexec.Multiplexed())
	if err != nil {
		t.Fatalf("exec container TUI: %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read container TUI output: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("container TUI exit code = %d, output = %q", exitCode, output)
	}
}

func execDovik(t *testing.T, ctx context.Context, container testcontainers.Container, arguments ...string) string {
	t.Helper()
	command := append([]string{"dovik"}, arguments...)
	exitCode, reader, err := container.Exec(ctx, command)
	if err != nil {
		t.Fatalf("exec %v: %v", command, err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read %v output: %v", command, err)
	}
	if exitCode != 0 {
		t.Fatalf("exec %v exit code = %d, output = %s", command, exitCode, output)
	}
	return string(output)
}
