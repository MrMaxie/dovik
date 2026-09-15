package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/MrMaxie/dovik/internal/identity"
)

type stubGHClient struct {
	snapshot      identity.Snapshot
	identityErr   error
	executeCalls  int
	lastRequest   identity.ExecutionRequest
	executeOutput string
}

func (client *stubGHClient) Identity(context.Context, identity.Request) (identity.Snapshot, error) {
	return client.snapshot, client.identityErr
}

func (client *stubGHClient) ExecuteGH(_ context.Context, request identity.ExecutionRequest, stdout, _ io.Writer) (int, error) {
	client.executeCalls++
	client.lastRequest = request
	output := client.executeOutput
	if output == "" {
		output = "governed output\n"
	}
	_, _ = io.WriteString(stdout, output)
	return 0, nil
}

func TestGHCredentialHelperForwardsGitProtocolInput(t *testing.T) {
	client := &stubGHClient{snapshot: identity.Snapshot{State: identity.State{Projects: []identity.Project{{
		ID: "project", Root: "project-root", Repository: "owner/repo", Persona: "persona",
		Mode: "proxy-level", Policy: identity.Policy{Preset: "maintain"}, ProxyEnabled: true,
	}}}}}
	input := "protocol=https\nhost=github.com\n\n"
	code, err := runGHWith(context.Background(), client, []string{"auth", "git-credential", "get"}, strings.NewReader(input), io.Discard, io.Discard,
		func(context.Context, string) (string, error) { return "project-root", nil },
		func(context.Context, string, []string, io.Reader, io.Writer, io.Writer, []string) (int, error) {
			t.Fatal("configured project bypassed governed execution")
			return 0, nil
		},
	)
	if err != nil || code != 0 || client.executeCalls != 1 || string(client.lastRequest.Invocation.Input) != input {
		t.Fatalf("code=%d err=%v execute=%d input=%q", code, err, client.executeCalls, client.lastRequest.Invocation.Input)
	}
}

func TestGHCredentialHelperIgnoresGitStoreNotification(t *testing.T) {
	client := &stubGHClient{snapshot: identity.Snapshot{State: identity.State{Projects: []identity.Project{{
		ID: "project", Root: "project-root", Repository: "owner/repo", Persona: "persona",
		Mode: "proxy-level", Policy: identity.Policy{Preset: "maintain"}, ProxyEnabled: true,
	}}}}}
	code, err := runGHWith(context.Background(), client, []string{"auth", "git-credential", "store"}, strings.NewReader("protocol=https\nhost=github.com\npassword=secret\n\n"), io.Discard, io.Discard,
		func(context.Context, string) (string, error) { return "project-root", nil },
		func(context.Context, string, []string, io.Reader, io.Writer, io.Writer, []string) (int, error) {
			t.Fatal("configured project delegated a credential store notification")
			return 0, nil
		},
	)
	if err != nil || code != 0 || client.executeCalls != 0 {
		t.Fatalf("code=%d err=%v execute=%d", code, err, client.executeCalls)
	}
}

func TestGHMetadataUsesOriginalCLIOutput(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"-h"}, {"help"}, {"--version"}, {"issue", "list", "--help"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			client := &stubGHClient{identityErr: errors.New("identity must not be inspected")}
			var stdout, stderr bytes.Buffer
			called := false
			code, err := runGHWith(context.Background(), client, args, strings.NewReader(""), &stdout, &stderr,
				func(context.Context, string) (string, error) {
					t.Fatal("metadata invocation inspected the repository")
					return "", nil
				},
				func(_ context.Context, path string, got []string, _ io.Reader, output, _ io.Writer, _ []string) (int, error) {
					called = true
					if path != "" || strings.Join(got, " ") != strings.Join(args, " ") {
						t.Fatalf("original invocation = %q %v", path, got)
					}
					_, _ = io.WriteString(output, "upstream GitHub CLI output\n")
					return 0, nil
				},
			)
			if err != nil || code != 0 || !called || stderr.Len() != 0 || strings.Contains(strings.ToLower(stdout.String()), "dovik") {
				t.Fatalf("code=%d err=%v called=%t stdout=%q stderr=%q", code, err, called, stdout.String(), stderr.String())
			}
		})
	}
}

func TestGHUnconfiguredProjectUsesOriginalCLI(t *testing.T) {
	client := &stubGHClient{snapshot: identity.Snapshot{State: identity.State{GHPath: "original-gh"}}}
	var stdout, stderr bytes.Buffer
	called := false
	code, err := runGHWith(context.Background(), client, []string{"issue", "list"}, strings.NewReader(""), &stdout, &stderr,
		func(context.Context, string) (string, error) { return "project-root", nil },
		func(_ context.Context, path string, args []string, _ io.Reader, output, _ io.Writer, _ []string) (int, error) {
			called = true
			if path != "original-gh" || strings.Join(args, " ") != "issue list" {
				t.Fatalf("original invocation = %q %v", path, args)
			}
			_, _ = io.WriteString(output, "[]\n")
			return 0, nil
		},
	)
	if err != nil || code != 0 || !called || client.executeCalls != 0 || stdout.String() != "[]\n" || stderr.Len() != 0 {
		t.Fatalf("code=%d err=%v called=%t execute=%d stdout=%q stderr=%q", code, err, called, client.executeCalls, stdout.String(), stderr.String())
	}
}

func TestGHUnavailableDaemonFallsBackToOriginalCLI(t *testing.T) {
	client := &stubGHClient{identityErr: errors.New("daemon unavailable")}
	called := false
	code, err := runGHWith(context.Background(), client, []string{"repo", "view"}, strings.NewReader(""), io.Discard, io.Discard,
		func(context.Context, string) (string, error) {
			t.Fatal("fallback inspected the repository")
			return "", nil
		},
		func(_ context.Context, path string, _ []string, _ io.Reader, _, _ io.Writer, _ []string) (int, error) {
			called = true
			if path != "" {
				t.Fatalf("fallback path = %q, want PATH discovery", path)
			}
			return 0, nil
		},
	)
	if err != nil || code != 0 || !called {
		t.Fatalf("code=%d err=%v called=%t", code, err, called)
	}
}

func TestGHConfiguredProjectUsesGovernedExecution(t *testing.T) {
	client := &stubGHClient{snapshot: identity.Snapshot{State: identity.State{Projects: []identity.Project{{
		ID: "project", Root: "project-root", Repository: "owner/repo", Persona: "persona",
		Mode: "proxy-level", Policy: identity.Policy{Preset: "read-only"}, ProxyEnabled: true,
	}}}}}
	code, err := runGHWith(context.Background(), client, []string{"issue", "list"}, strings.NewReader(""), io.Discard, io.Discard,
		func(context.Context, string) (string, error) { return "project-root", nil },
		func(context.Context, string, []string, io.Reader, io.Writer, io.Writer, []string) (int, error) {
			t.Fatal("configured project bypassed governed execution")
			return 0, nil
		},
	)
	if err != nil || code != 0 || client.executeCalls != 1 {
		t.Fatalf("code=%d err=%v execute=%d", code, err, client.executeCalls)
	}
}

func TestGHInteractivePRCreateUsesSelectedPersonaInCurrentTerminal(t *testing.T) {
	t.Setenv("GH_TOKEN", "wrong")
	t.Setenv("GITHUB_TOKEN", "wrong")
	client := &stubGHClient{
		executeOutput: "username=x-access-token\npassword=selected-token\n\n",
		snapshot: identity.Snapshot{State: identity.State{
			GHPath:   "original-gh",
			Personas: []identity.Persona{{ID: "work", Host: "github.com", Account: "example"}},
			Projects: []identity.Project{{
				ID: "project", Root: "project-root", Repository: "owner/repo", Persona: "work",
				Mode: "proxy-level", Policy: identity.Policy{Preset: "collaborate"}, ProxyEnabled: true,
			}},
		}},
	}
	called := false
	code, err := runGHWith(context.Background(), client, []string{"pr", "create"}, strings.NewReader("terminal input"), io.Discard, io.Discard,
		func(context.Context, string) (string, error) { return "project-root", nil },
		func(_ context.Context, path string, args []string, input io.Reader, _, _ io.Writer, environment []string) (int, error) {
			called = true
			if path != "original-gh" || strings.Join(args, " ") != "pr create --repo github.com/owner/repo" {
				t.Fatalf("unexpected original invocation: path=%q args=%q", path, args)
			}
			values := map[string]string{}
			for _, item := range environment {
				key, value, _ := strings.Cut(item, "=")
				values[strings.ToUpper(key)] = value
			}
			if values["GH_TOKEN"] != "selected-token" || values["GITHUB_TOKEN"] != "" || values["GH_PROMPT_DISABLED"] != "" {
				t.Fatal("interactive environment did not select the configured account")
			}
			data, _ := io.ReadAll(input)
			if string(data) != "terminal input" {
				t.Fatalf("terminal input was not preserved: %q", data)
			}
			return 0, nil
		},
	)
	if err != nil || code != 0 || !called || client.executeCalls != 1 {
		t.Fatalf("code=%d err=%v called=%t execute=%d", code, err, called, client.executeCalls)
	}
}
