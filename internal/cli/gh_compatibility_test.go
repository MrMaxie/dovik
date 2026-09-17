package cli

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/MrMaxie/dovik/internal/identity"
)

// Captured from `gh help reference` in GitHub CLI 2.98.0. Ordinary proxy
// execution is deliberately not implemented as this version-specific list;
// the inventory only proves compatibility with the complete observed surface.
const ghCLI298Commands = `
agent-task create
agent-task list
agent-task view
alias delete
alias import
alias list
alias set
api
attestation download
attestation trusted-root
attestation verify
auth login
auth logout
auth refresh
auth setup-git
auth status
auth switch
auth token
browse
cache delete
cache list
co
codespace code
codespace cp
codespace create
codespace delete
codespace edit
codespace jupyter
codespace list
codespace logs
codespace ports forward
codespace ports visibility
codespace rebuild
codespace ssh
codespace stop
codespace view
completion
config clear-cache
config get
config list
config set
copilot
discussion comment
discussion create
discussion edit
discussion list
discussion view
extension browse
extension create
extension exec
extension install
extension list
extension remove
extension search
extension upgrade
gist clone
gist create
gist delete
gist edit
gist list
gist rename
gist view
gpg-key add
gpg-key delete
gpg-key list
issue close
issue comment
issue create
issue delete
issue develop
issue edit
issue list
issue lock
issue pin
issue reopen
issue status
issue transfer
issue unlock
issue unpin
issue view
label clone
label create
label delete
label edit
label list
licenses
org list
pr checkout
pr checks
pr close
pr comment
pr create
pr diff
pr edit
pr list
pr lock
pr merge
pr ready
pr reopen
pr revert
pr review
pr status
pr unlock
pr update-branch
pr view
preview prompter
project close
project copy
project create
project delete
project edit
project field-create
project field-delete
project field-list
project item-add
project item-archive
project item-create
project item-delete
project item-edit
project item-list
project link
project list
project mark-template
project unlink
project view
release create
release delete
release delete-asset
release download
release edit
release list
release upload
release verify
release verify-asset
release view
repo archive
repo autolink create
repo autolink delete
repo autolink list
repo autolink view
repo clone
repo create
repo delete
repo deploy-key add
repo deploy-key delete
repo deploy-key list
repo edit
repo fork
repo gitignore list
repo gitignore view
repo license list
repo license view
repo list
repo read-dir
repo read-file
repo rename
repo set-default
repo sync
repo unarchive
repo view
ruleset check
ruleset list
ruleset view
run cancel
run delete
run download
run list
run rerun
run view
run watch
search code
search commits
search issues
search prs
search repos
secret delete
secret list
secret set
skill install
skill list
skill preview
skill publish
skill search
skill update
ssh-key add
ssh-key delete
ssh-key list
stack
status
variable delete
variable get
variable list
variable set
workflow disable
workflow enable
workflow list
workflow run
workflow view
`

func TestGHCLI298CommandSurfaceUsesTransparentProxy(t *testing.T) {
	commands := strings.Split(strings.TrimSpace(ghCLI298Commands), "\n")
	if len(commands) != 198 {
		t.Fatalf("inventory has %d commands, want 198", len(commands))
	}
	roots := map[string]bool{}
	for _, command := range commands {
		roots[strings.Fields(command)[0]] = true
	}
	if len(roots) != 35 {
		t.Fatalf("inventory has %d root groups, want 35", len(roots))
	}

	blocked := map[string]bool{
		"auth login": true, "auth logout": true, "auth refresh": true,
		"auth setup-git": true, "auth switch": true, "auth token": true,
	}
	for _, command := range commands {
		command := command
		t.Run(strings.ReplaceAll(command, " ", "_"), func(t *testing.T) {
			client := transparentGHClient()
			called := false
			args := strings.Fields(command)
			code, err := runGHWith(context.Background(), client, args, strings.NewReader("input"), io.Discard, io.Discard,
				func(context.Context, string) (string, error) { return "project-root", nil },
				func(_ context.Context, path string, got []string, _ io.Reader, _, _ io.Writer, environment []string) (int, error) {
					called = true
					if path != "original-gh" || strings.Join(got, " ") != command {
						t.Fatalf("original invocation = %q %q", path, got)
					}
					if !environmentContains(environment, "GH_TOKEN=selected-token") {
						t.Fatal("configured persona token was not selected")
					}
					return 0, nil
				},
			)
			if blocked[command] {
				if err == nil || code != 1 || called || client.executeCalls != 0 {
					t.Fatalf("blocked command: code=%d err=%v called=%t execute=%d", code, err, called, client.executeCalls)
				}
				return
			}
			if err != nil || code != 0 || !called || client.executeCalls != 1 {
				t.Fatalf("passthrough: code=%d err=%v called=%t execute=%d", code, err, called, client.executeCalls)
			}
		})
	}
}

func TestGHUnknownCommandPassesThrough(t *testing.T) {
	client := transparentGHClient()
	args := []string{"future-command", "subcommand", "--new-flag", "value"}
	called := false
	var stdout, stderr strings.Builder
	code, err := runGHWith(context.Background(), client, args, strings.NewReader("future input"), &stdout, &stderr,
		func(context.Context, string) (string, error) { return "project-root", nil },
		func(_ context.Context, _ string, got []string, input io.Reader, output, errors io.Writer, _ []string) (int, error) {
			called = true
			if strings.Join(got, " ") != strings.Join(args, " ") {
				t.Fatalf("arguments changed: %q", got)
			}
			data, _ := io.ReadAll(input)
			if string(data) != "future input" {
				t.Fatalf("stdin changed: %q", data)
			}
			_, _ = io.WriteString(output, "future output")
			_, _ = io.WriteString(errors, "future error")
			return 42, nil
		},
	)
	if err != nil || code != 42 || !called || client.executeCalls != 1 || stdout.String() != "future output" || stderr.String() != "future error" {
		t.Fatalf("code=%d err=%v called=%t execute=%d stdout=%q stderr=%q", code, err, called, client.executeCalls, stdout.String(), stderr.String())
	}
}

func TestGHSafeAuthStatusPassesThrough(t *testing.T) {
	client := transparentGHClient()
	args := []string{"auth", "status", "--active", "--hostname", "github.com"}
	called := false
	code, err := runGHWith(context.Background(), client, args, strings.NewReader(""), io.Discard, io.Discard,
		func(context.Context, string) (string, error) { return "project-root", nil },
		func(_ context.Context, _ string, got []string, _ io.Reader, _, _ io.Writer, _ []string) (int, error) {
			called = true
			if strings.Join(got, " ") != strings.Join(args, " ") {
				t.Fatalf("arguments changed: %q", got)
			}
			return 0, nil
		},
	)
	if err != nil || code != 0 || !called || client.executeCalls != 1 {
		t.Fatalf("code=%d err=%v called=%t execute=%d", code, err, called, client.executeCalls)
	}
}

func TestGHAuthenticationGuardRunsBeforeCredentialAccess(t *testing.T) {
	for _, args := range [][]string{
		{"auth", "login"},
		{"auth", "logout"},
		{"auth", "refresh"},
		{"auth", "setup-git"},
		{"auth", "switch"},
		{"auth", "token"},
		{"auth", "status", "--show-token"},
		{"auth", "status", "--show-token=true"},
		{"auth", "status", "-t"},
		{"auth", "status", "-at"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			client := transparentGHClient()
			code, err := runGHWith(context.Background(), client, args, strings.NewReader(""), io.Discard, io.Discard,
				func(context.Context, string) (string, error) { return "project-root", nil },
				func(context.Context, string, []string, io.Reader, io.Writer, io.Writer, []string) (int, error) {
					t.Fatal("blocked authentication command reached original gh")
					return 0, nil
				},
			)
			if err == nil || code != 1 || client.executeCalls != 0 {
				t.Fatalf("code=%d err=%v execute=%d", code, err, client.executeCalls)
			}
		})
	}
}

func transparentGHClient() *stubGHClient {
	return &stubGHClient{
		executeOutput: "username=x-access-token\npassword=selected-token\n\n",
		snapshot: identity.Snapshot{State: identity.State{
			GHPath:   "original-gh",
			Personas: []identity.Persona{{ID: "persona", Host: "github.com", Account: "example"}},
			Projects: []identity.Project{{
				ID: "project", Root: "project-root", Repository: "owner/repo", Persona: "persona",
				Mode: "proxy-level", Policy: identity.Policy{Preset: "read-only"}, ProxyEnabled: true,
			}},
		}},
	}
}
