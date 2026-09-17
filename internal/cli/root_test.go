package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"charm.land/huh/v2"
	"github.com/MrMaxie/dovik/internal/identity"
)

type daemonLauncherStub struct {
	starts int
	start  func() (int, error)
}

func (launcher *daemonLauncherStub) Start() (int, error) {
	launcher.starts++
	return launcher.start()
}

func TestRootMenuOptionsFollowRepositoryState(t *testing.T) {
	configured := true
	unconfigured := false
	tests := []struct {
		name       string
		root       string
		configured *bool
		labels     []string
	}{
		{name: "outside repository", labels: []string{"Help", "Open TUI"}},
		{name: "configured", root: "repo", configured: &configured, labels: []string{"Help", "Edit this repository", "Open TUI"}},
		{name: "unconfigured", root: "repo", configured: &unconfigured, labels: []string{"Help", "Configure this repository", "Open TUI"}},
		{name: "daemon unavailable", root: "repo", labels: []string{"Help", "Configure project", "Open TUI"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := rootMenuOptions(test.root, test.configured)
			if len(options) != len(test.labels) {
				t.Fatalf("options = %#v", options)
			}
			for index, label := range test.labels {
				if options[index].Key != label {
					t.Fatalf("option %d = %q, want %q", index, options[index].Key, label)
				}
			}
			if options[0].Value != rootActionHelp {
				t.Fatalf("default option = %q", options[0].Value)
			}
		})
	}
}

func TestBareRedirectedDovikPrintsHelpAndSucceeds(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunIO(context.Background(), nil, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestBareJSONStillRequiresCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunIO(context.Background(), []string{"--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "command is required") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestRootMenuCancelIsNotAnError(t *testing.T) {
	if !rootMenuCanceled(huh.ErrUserAborted) {
		t.Fatal("user abort was not treated as menu cancellation")
	}
	if rootMenuCanceled(context.Canceled) {
		t.Fatal("context cancellation was mistaken for a user abort")
	}
}

func TestConfigurationStartsUnavailableDaemonAndWaitsForIdentity(t *testing.T) {
	client := &identityStubClient{err: errors.New("offline")}
	launcher := &daemonLauncherStub{start: func() (int, error) {
		client.err = nil
		client.snapshot = identity.Snapshot{State: identity.State{}}
		return 42, nil
	}}
	if err := ensureIdentityDaemon(context.Background(), client, launcher); err != nil {
		t.Fatal(err)
	}
	if launcher.starts != 1 || client.calls < 2 {
		t.Fatalf("starts = %d, identity calls = %d", launcher.starts, client.calls)
	}
}
