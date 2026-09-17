package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/operatorclient"
)

type identityStubClient struct {
	operatorclient.Client
	snapshot identity.Snapshot
	err      error
	wait     bool
	calls    int
}

func (client *identityStubClient) Identity(ctx context.Context, _ identity.Request) (identity.Snapshot, error) {
	client.calls++
	if client.wait {
		<-ctx.Done()
		return identity.Snapshot{}, ctx.Err()
	}
	return client.snapshot, client.err
}

func TestWhoAmIReportsConfiguredAndUnconfiguredRepositories(t *testing.T) {
	t.Setenv("DOVIK_AGENT_ENDPOINT", "")
	root := initGitRepository(t)
	withWorkingDirectory(t, root)

	configured := &identityStubClient{snapshot: identity.Snapshot{State: identity.State{
		Personas: []identity.Persona{{ID: "personal", Name: "Personal"}},
		Projects: []identity.Project{{ID: "project", Root: root, Persona: "personal"}},
	}}}
	var stdout bytes.Buffer
	if err := runWhoAmI(context.Background(), configured, &stdout, outputMode{}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Personal\n" {
		t.Fatalf("configured output = %q", stdout.String())
	}

	stdout.Reset()
	unconfigured := &identityStubClient{snapshot: identity.Snapshot{State: identity.State{}}}
	if err := runWhoAmI(context.Background(), unconfigured, &stdout, outputMode{}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "?\n" {
		t.Fatalf("unconfigured output = %q", stdout.String())
	}
}

func TestWhoAmIReportsNonRepositoryWithoutDaemon(t *testing.T) {
	t.Setenv("DOVIK_AGENT_ENDPOINT", "")
	withWorkingDirectory(t, t.TempDir())
	client := &identityStubClient{err: errors.New("must not be called")}
	var stdout bytes.Buffer
	if err := runWhoAmI(context.Background(), client, &stdout, outputMode{}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "?\n" || client.calls != 0 {
		t.Fatalf("output = %q, daemon calls = %d", stdout.String(), client.calls)
	}
}

func TestWhoAmIReportsUnavailableDaemonAndTimeout(t *testing.T) {
	t.Setenv("DOVIK_AGENT_ENDPOINT", "")
	root := initGitRepository(t)
	withWorkingDirectory(t, root)
	for _, client := range []*identityStubClient{{err: errors.New("offline")}, {wait: true}} {
		var stdout bytes.Buffer
		if err := runWhoAmI(context.Background(), client, &stdout, outputMode{}); err != nil {
			t.Fatal(err)
		}
		if stdout.String() != "!\n" {
			t.Fatalf("unavailable output = %q", stdout.String())
		}
	}
}

func TestWhoAmIJSONStates(t *testing.T) {
	t.Setenv("DOVIK_AGENT_ENDPOINT", "")
	configuredRoot := initGitRepository(t)
	unconfiguredRoot := initGitRepository(t)
	nonRepository := t.TempDir()
	configuredClient := &identityStubClient{snapshot: identity.Snapshot{State: identity.State{
		Personas: []identity.Persona{{ID: "work", Name: "Work"}},
		Projects: []identity.Project{{ID: "project", Root: configuredRoot, Persona: "work"}},
	}}}
	tests := []struct {
		name       string
		directory  string
		client     *identityStubClient
		state      string
		configured *bool
		persona    *string
	}{
		{name: "configured", directory: configuredRoot, client: configuredClient, state: "configured", configured: boolValue(true), persona: stringValue("Work")},
		{name: "unconfigured", directory: unconfiguredRoot, client: &identityStubClient{snapshot: identity.Snapshot{State: identity.State{}}}, state: "unconfigured", configured: boolValue(false)},
		{name: "not repository", directory: nonRepository, client: &identityStubClient{}, state: "not_repository", configured: boolValue(false)},
		{name: "unavailable", directory: unconfiguredRoot, client: &identityStubClient{err: errors.New("offline")}, state: "unavailable"},
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(previous) }()
	for _, test := range tests {
		if err := os.Chdir(test.directory); err != nil {
			t.Fatal(err)
		}
		var stdout bytes.Buffer
		if err := runWhoAmI(context.Background(), test.client, &stdout, outputMode{json: true}); err != nil {
			t.Fatal(err)
		}
		var result whoamiResult
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.State != test.state || !equalBoolPointers(result.Configured, test.configured) || !equalStringPointers(result.Persona, test.persona) {
			t.Fatalf("%s result = %#v", test.name, result)
		}
	}
}

func boolValue(value bool) *bool       { return &value }
func stringValue(value string) *string { return &value }

func equalBoolPointers(left, right *bool) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func equalStringPointers(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func TestWhoAmIUsesRestrictedSessionContext(t *testing.T) {
	t.Setenv("DOVIK_AGENT_ENDPOINT", "test-agent")
	previousDial := dialAgentContext
	t.Cleanup(func() { dialAgentContext = previousDial })
	dialAgentContext = func(context.Context, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			line, err := bufio.NewReader(server).ReadBytes('\n')
			if err != nil {
				return
			}
			var request identity.AgentRequest
			if err := json.Unmarshal(line, &request); err != nil || request.Action != "context" {
				return
			}
			_ = json.NewEncoder(server).Encode(identity.Frame{Context: &identity.AgentContext{Persona: identity.Persona{Name: "Isolated"}}})
		}()
		return client, nil
	}
	client := &identityStubClient{err: errors.New("operator client must not be called")}
	var stdout bytes.Buffer
	if err := runWhoAmI(context.Background(), client, &stdout, outputMode{}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Isolated\n" || client.calls != 0 {
		t.Fatalf("output = %q, operator calls = %d", stdout.String(), client.calls)
	}
}

func TestWhoAmIRejectsArguments(t *testing.T) {
	t.Setenv("DOVIK_AGENT_ENDPOINT", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunIO(context.Background(), []string{"whoami", "extra"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "accepts no arguments") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func initGitRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	command := exec.Command("git", "init", "--quiet", root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func withWorkingDirectory(t *testing.T, directory string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
}
