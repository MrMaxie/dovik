package identity

import (
	"strings"
	"testing"
)

func TestCatalogueRejectsEscapes(t *testing.T) {
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "maintain"}}
	persona := Persona{Host: "github.com"}
	attacks := [][]string{
		{"auth", "token"}, {"auth", "status", "--show-token"}, {"alias", "set", "evil", "!sh"},
		{"pr", "view", "1", "--repo", "other/repo"}, {"pr", "view", "https://evil.test/1"},
		{"pr", "view", "1", "--web"}, {"pr", "view", "1", "--unknown"},
		{"pr", "view", "1", "--", "another"},
		{"pr", "create", "--fill"}, {"pr", "comment", "1", "--body-file", "/private/token"},
		{"pr", "merge", "1", "--delete-branch"}, {"api", "graphql"},
		{"api", "repos/owner/repo/../other"}, {"api", "repos/owner/repo", "-H", "Authorization: bad"},
		{"api", "repos/owner/repo", "-F", "x=@/private/token"},
		{"api", "repos/owner/repo/issues/1", "-X", "PATCH", "-f", "state=closed"},
	}
	for _, args := range attacks {
		if _, err := Authorize(p, persona, Invocation{Arguments: args}); err == nil {
			t.Errorf("accepted %q", args)
		}
	}
}

func TestCloseCommentRequiresBothPermissions(t *testing.T) {
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "maintain", Exceptions: map[string]bool{"comment": false}}}
	if _, err := Authorize(p, Persona{Host: "github.com"}, Invocation{Arguments: []string{"issue", "close", "1", "--comment", "body"}}); err == nil {
		t.Fatal("close bypassed comment denial")
	}
}

func TestCatalogueInjectsBoundRepositoryAndHonorsExceptions(t *testing.T) {
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "collaborate"}}
	persona := Persona{Host: "github.com"}
	command, err := Authorize(p, persona, Invocation{Arguments: []string{"pr", "view", "12", "--json", "title,body"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(command.Arguments, " "), "--repo github.com/owner/repo") {
		t.Fatal(command)
	}
	if _, err := Authorize(p, persona, Invocation{Arguments: []string{"pr", "merge", "12", "--squash"}}); err == nil {
		t.Fatal("merge unexpectedly permitted")
	}
	p.Policy.Exceptions = map[string]bool{"merge": true}
	if _, err := Authorize(p, persona, Invocation{Arguments: []string{"pr", "merge", "12", "--squash"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Authorize(p, persona, Invocation{Arguments: []string{"api", "repos/owner/repo/issues", "-f", "state=open"}}); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogueAcceptsInteractivePRCreate(t *testing.T) {
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "collaborate"}}
	persona := Persona{Host: "github.com"}
	command, err := Authorize(p, persona, Invocation{Arguments: []string{"pr", "create"}})
	if err != nil {
		t.Fatal(err)
	}
	if !command.Interactive || strings.Join(command.Arguments, " ") != "pr create --repo github.com/owner/repo" {
		t.Fatalf("unexpected interactive command: %#v", command)
	}
}

func TestCatalogueAcceptsBoundGitCredentialLookup(t *testing.T) {
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "maintain"}}
	persona := Persona{Host: "github.com"}
	command, err := Authorize(p, persona, Invocation{
		Arguments: []string{"auth", "git-credential", "get"},
		Input:     []byte("capability[]=authtype\ncapability[]=state\nprotocol=https\nhost=github.com\npath=owner/repo.git\nwwwauth[]=Basic realm=\"GitHub\"\n\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !command.Credential {
		t.Fatal("credential lookup was not marked as credential output")
	}
}

func TestCatalogueRejectsUnboundGitCredentialRequests(t *testing.T) {
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "maintain"}}
	persona := Persona{Host: "github.com"}
	requests := []Invocation{
		{Arguments: []string{"auth", "git-credential"}, Input: []byte("protocol=https\nhost=github.com\n\n")},
		{Arguments: []string{"auth", "git-credential", "store"}, Input: []byte("protocol=https\nhost=github.com\n\n")},
		{Arguments: []string{"auth", "git-credential", "get"}, Input: []byte("protocol=http\nhost=github.com\n\n")},
		{Arguments: []string{"auth", "git-credential", "get"}, Input: []byte("protocol=https\nhost=evil.test\n\n")},
		{Arguments: []string{"auth", "git-credential", "get"}, Input: []byte("protocol=https\nhost=github.com\npath=other/repo\n\n")},
		{Arguments: []string{"auth", "git-credential", "get"}, Input: []byte("protocol=https\nhost=github.com\npassword=secret\n\n")},
		{Arguments: []string{"auth", "git-credential", "get"}, Input: []byte("capability[]=unknown\nprotocol=https\nhost=github.com\n\n")},
	}
	for _, request := range requests {
		if _, err := Authorize(p, persona, request); err == nil {
			t.Errorf("accepted %#v", request)
		}
	}
}
