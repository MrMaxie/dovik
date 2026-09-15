package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"
)

type GitHubAccount struct {
	Host    string `json:"host"`
	Account string `json:"account"`
	Active  bool   `json:"active"`
}

// GitHubAccounts asks the original CLI for account metadata, never tokens.
func GitHubAccounts(ctx context.Context, gh string) ([]GitHubAccount, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	path, err := ValidateGH(gh)
	if err != nil {
		return nil, err
	}
	command := BackgroundCommand(ctx, path, "auth", "status", "--json", "hosts", "--jq", `.hosts | to_entries | map(. as $h | .value[] | {host: $h.key, account: .login, active: .active})`)
	command.Env = operatorEnvironment()
	command.Stderr = io.Discard
	var output limitedBuffer
	output.limit = 256 * 1024
	command.Stdout = &output
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("could not list GitHub CLI accounts")
	}
	return parseAccounts(output.Bytes())
}

func parseAccounts(data []byte) ([]GitHubAccount, error) {
	var accounts []GitHubAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, fmt.Errorf("invalid GitHub CLI account list")
	}
	result := []GitHubAccount{}
	seen := map[string]bool{}
	for _, account := range accounts {
		key := strings.ToLower(account.Host + "/" + account.Account)
		if !hostName.MatchString(account.Host) || !identifier.MatchString(account.Account) || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, account)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Active != result[j].Active {
			return result[i].Active
		}
		return result[i].Host+"/"+result[i].Account < result[j].Host+"/"+result[j].Account
	})
	return result, nil
}

type GitDefaults struct {
	Host       string
	Repository string
	Name       string
	Email      string
}

// DiscoverGitDefaults reads the checkout's origin and effective Git author.
func DiscoverGitDefaults(ctx context.Context, root string) GitDefaults {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	read := func(args ...string) string {
		command := BackgroundCommand(ctx, "git", append([]string{"-C", root}, args...)...)
		output, err := command.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(output))
	}
	host, repository := parseGitRemote(read("remote", "get-url", "origin"))
	return GitDefaults{Host: host, Repository: repository, Name: read("config", "--get", "user.name"), Email: read("config", "--get", "user.email")}
}

func parseGitRemote(remote string) (string, string) {
	var host, path string
	if strings.Contains(remote, "://") {
		parsed, err := url.Parse(remote)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http" && parsed.Scheme != "ssh") {
			return "", ""
		}
		host, path = parsed.Hostname(), parsed.Path
	} else {
		left, right, ok := strings.Cut(remote, ":")
		if !ok || !strings.Contains(left, "@") {
			return "", ""
		}
		_, host, _ = strings.Cut(left, "@")
		path = right
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if !hostName.MatchString(host) || len(parts) != 2 || !identifier.MatchString(parts[0]) || !identifier.MatchString(parts[1]) {
		return "", ""
	}
	return host, path
}
