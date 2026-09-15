package identity

import (
	"context"
	"testing"
)

func TestParseGitRemote(t *testing.T) {
	for _, remote := range []string{"git@github.com:owner/repo.git", "https://github.com/owner/repo.git", "ssh://git@github.com/owner/repo.git", "https://github.com/owner/repo"} {
		host, repo := parseGitRemote(remote)
		if host != "github.com" || repo != "owner/repo" {
			t.Fatalf("%q: %q %q", remote, host, repo)
		}
	}
	for _, remote := range []string{"C:/repos/project", "../project", "file:///owner/repo", "https://github.com/owner/repo/extra"} {
		if _, repo := parseGitRemote(remote); repo != "" {
			t.Fatalf("accepted %q", remote)
		}
	}
}

func TestDiscoverGitDefaultsFromOrigin(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"init", "--quiet"}, {"remote", "add", "origin", "git@github.com:owner/repo.git"}, {"config", "user.name", "Local Author"}, {"config", "user.email", "author@example.test"}} {
		if out, err := BackgroundCommand(context.Background(), "git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	got := DiscoverGitDefaults(context.Background(), root)
	if got.Host != "github.com" || got.Repository != "owner/repo" || got.Name != "Local Author" || got.Email != "author@example.test" {
		t.Fatalf("%+v", got)
	}
}

func TestParseAccountsIncludesInactiveAndDeduplicates(t *testing.T) {
	accounts, err := parseAccounts([]byte(`[{"host":"github.com","account":"work","active":false},{"host":"github.com","account":"personal","active":true},{"host":"github.com","account":"WORK"},{"host":"github.com","account":"bad\nname"}]`))
	if err != nil || len(accounts) != 2 || accounts[0].Account != "personal" || accounts[1].Account != "work" {
		t.Fatalf("%+v %v", accounts, err)
	}
}
