package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/supervision"
)

func TestVersionAndHelpDoNotRequireDaemon(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "version", arguments: []string{"--version"}, contains: "dovik 1.0.0"},
		{name: "root help", arguments: []string{"--help"}, contains: "Usage:"},
		{name: "command help", arguments: []string{"--endpoint", "relative-is-invalid-for-a-daemon", "process", "--help"}, contains: "dovik process"},
		{name: "subcommand help", arguments: []string{"process", "start", "--help"}, contains: "process start"},
		{name: "proxy help", arguments: []string{"gh", "--help"}, contains: "dovik gh --"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := RunIO(context.Background(), test.arguments, strings.NewReader(""), &stdout, &stderr); code != 0 {
				t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.contains) {
				t.Fatalf("stdout = %q, want substring %q", stdout.String(), test.contains)
			}
		})
	}
}

const cliHelperEnvironment = "DOVIK_CLI_HELPER"

func TestCLIRegistryAndLifecycleCommands(t *testing.T) {
	endpoint, stopServer := startCLITestServer(t)
	defer stopServer()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	root := t.TempDir()

	runCLI(t, ctx, endpoint, "project", "add", "--id", "project", "--root", root)
	runCLI(t, ctx, endpoint,
		"process", "add",
		"--project", "project",
		"--id", "api",
		"--command", os.Args[0],
		"--arg=-test.run=TestCLIHelperProcess",
		"--env", cliHelperEnvironment+"=1",
		"--env", "SECRET=private",
	)

	startOutput := runCLI(t, ctx, endpoint, "process", "start", "--project", "project", "--process", "api")
	if !strings.Contains(startOutput, "\trunning\t") {
		t.Fatalf("start output = %q, want running state", startOutput)
	}
	stopOutput := runCLI(t, ctx, endpoint, "process", "stop", "--project", "project", "--process", "api")
	if !strings.Contains(stopOutput, "\tstopped\t") {
		t.Fatalf("stop output = %q, want stopped state", stopOutput)
	}
	listOutput := runCLI(t, ctx, endpoint, "process", "list", "--project", "project")
	if strings.Contains(listOutput, "private") || !strings.Contains(listOutput, "api") {
		t.Fatalf("process list output = %q, want redacted process", listOutput)
	}
}

func TestCLIRejectsInvalidEnvironmentOverride(t *testing.T) {
	endpoint, stopServer := startCLITestServer(t)
	defer stopServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(ctx, []string{
		"--endpoint", endpoint,
		"process", "add",
		"--project", "project",
		"--id", "api",
		"--command", os.Args[0],
		"--env", "PRIVATE_VALUE_WITHOUT_SEPARATOR",
	}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "want KEY=VALUE") || strings.Contains(stderr.String(), "PRIVATE_VALUE") {
		t.Fatalf("code = %d, stderr = %q, want validation error", code, stderr.String())
	}
}

func TestCLIJSONOutputIsSingleDocumentAndRedactsEnvironment(t *testing.T) {
	endpoint, stopServer := startCLITestServer(t)
	defer stopServer()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root := t.TempDir()

	runCLI(t, ctx, endpoint, "project", "add", "--id", "project", "--root", root)
	runCLI(t, ctx, endpoint,
		"process", "add",
		"--project", "project",
		"--id", "api",
		"--command", os.Args[0],
		"--env", "SECRET=private",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(ctx, []string{"--endpoint", endpoint, "--json", "process", "list", "--project", "project"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	output := append([]byte(nil), stdout.Bytes()...)
	var processes []processJSON
	if err := json.Unmarshal(output, &processes); err != nil {
		t.Fatalf("decode JSON output: %v; output = %q", err, output)
	}
	if len(processes) != 1 {
		t.Fatalf("processes = %d, want 1; output = %q", len(processes), output)
	}
	if strings.Contains(string(output), "private") || strings.Contains(string(output), "environment") {
		t.Fatalf("JSON output exposed environment data: %s", output)
	}
	if !strings.Contains(string(output), `"projectId":"project"`) || !strings.Contains(string(output), `"id":"api"`) {
		t.Fatalf("JSON output = %q, want public process fields", output)
	}
}

func TestCLIJSONErrorUsesStderrAndMeaningfulExitCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(ctx, []string{"--json", "unknown"}, &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, want argument failure", code, stdout.String())
	}
	var result errorEnvelope
	if err := json.Unmarshal(stderr.Bytes(), &result); err != nil {
		t.Fatalf("decode structured error: %v; output = %q", err, stderr.String())
	}
	if result.Error.Code != "invalid_arguments" || result.Error.Message == "" {
		t.Fatalf("error = %#v, want invalid_arguments", result.Error)
	}
}

func TestCLITUIRejectsRedirectedStreams(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunIO(context.Background(), []string{"tui"}, strings.NewReader("q"), &stdout, &stderr)
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "interactive terminal") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestProductionTUIDaemonLauncherRequiresNativeWindowsDefaultEndpoint(t *testing.T) {
	launcher, err := productionTUIDaemonLauncher("custom", "default")
	if err != nil || launcher != nil {
		t.Fatalf("custom endpoint launcher = %T, err = %v, want nil", launcher, err)
	}
	launcher, err = productionTUIDaemonLauncher("default", "default")
	if err != nil {
		t.Fatalf("default endpoint launcher error = %v", err)
	}
	if runtime.GOOS != "windows" && launcher != nil {
		t.Fatalf("non-Windows launcher = %T, want nil", launcher)
	}
}

func runCLI(t *testing.T, ctx context.Context, endpoint string, arguments ...string) string {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	arguments = append([]string{"--endpoint", endpoint}, arguments...)
	if code := Run(ctx, arguments, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) code = %d, stderr = %q", arguments, code, stderr.String())
	}
	return stdout.String()
}

func startCLITestServer(t *testing.T) (string, func()) {
	t.Helper()
	var endpoint string
	if runtime.GOOS == "windows" {
		endpoint = fmt.Sprintf(`\\.\pipe\dovik-cli-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	} else {
		endpoint = filepath.Join(t.TempDir(), "state", "control.sock")
	}
	listener, err := control.ListenLocal(endpoint)
	if err != nil {
		t.Fatalf("ListenLocal() error = %v", err)
	}
	store := supervision.NewFileRegistry(filepath.Join(t.TempDir(), "state", "registry.json"))
	manager, err := supervision.NewLifecycleManager(supervision.NewRegistry(), store, 1024, 100)
	if err != nil {
		listener.Close()
		t.Fatalf("NewLifecycleManager() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- control.NewServer(manager).Serve(ctx, listener) }()
	return endpoint, func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	}
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv(cliHelperEnvironment) != "1" {
		return
	}
	_, _ = os.Stdout.WriteString("ready\n")
	for {
		time.Sleep(time.Second)
	}
}
