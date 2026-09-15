package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func GitRoot(ctx context.Context, path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	output, err := BackgroundCommand(ctx, "git", "-C", absolute, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("run this command in a Git repository or supply --root")
	}
	return filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(string(output))))
}

// ApplyGitAuthor runs in the caller's security context, never in the credential
// executor. Restore both local values if either write fails.
func ApplyGitAuthor(ctx context.Context, root string, p Persona) error {
	if err := p.Validate(); err != nil {
		return err
	}
	root, err := GitRoot(ctx, root)
	if err != nil {
		return err
	}
	keys := []string{"user.name", "user.email"}
	next := []string{p.GitName, p.GitEmail}
	previous := []string{"", ""}
	exists := []bool{false, false}
	for i, key := range keys {
		output, err := BackgroundCommand(ctx, "git", "-C", root, "config", "--local", "--get", key).Output()
		if err == nil {
			exists[i] = true
			previous[i] = strings.TrimSuffix(strings.TrimSuffix(string(output), "\n"), "\r")
		} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
			return fmt.Errorf("cannot inspect local Git author")
		}
	}
	for i, key := range keys {
		if err := BackgroundCommand(ctx, "git", "-C", root, "config", "--local", key, next[i]).Run(); err != nil {
			restore, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for j, k := range keys {
				if exists[j] {
					_ = BackgroundCommand(restore, "git", "-C", root, "config", "--local", k, previous[j]).Run()
				} else {
					_ = BackgroundCommand(restore, "git", "-C", root, "config", "--local", "--unset-all", k).Run()
				}
			}
			return fmt.Errorf("could not update local Git author; inspect git config --local")
		}
	}
	return nil
}

// ApplyGitCredentialHelper makes normal Git operations use the gh command from
// this repository. The empty entry clears inherited helpers for this checkout.
func ApplyGitCredentialHelper(ctx context.Context, root string) error {
	root, err := GitRoot(ctx, root)
	if err != nil {
		return err
	}
	previous, err := localGitConfigValues(ctx, root, "credential.helper")
	if err != nil {
		return err
	}
	if err := BackgroundCommand(ctx, "git", "-C", root, "config", "--local", "--replace-all", "credential.helper", "").Run(); err != nil {
		return fmt.Errorf("could not reset the repository-local Git credential helper")
	}
	if err := BackgroundCommand(ctx, "git", "-C", root, "config", "--local", "--add", "credential.helper", "!gh auth git-credential").Run(); err != nil {
		restore, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		restoreLocalGitConfigValues(restore, root, "credential.helper", previous)
		return fmt.Errorf("could not configure the repository-local Git credential helper")
	}
	return nil
}

func localGitConfigValues(ctx context.Context, root, key string) ([]string, error) {
	output, err := BackgroundCommand(ctx, "git", "-C", root, "config", "--local", "--null", "--get-all", key).Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot inspect repository-local Git configuration")
	}
	values := strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
	return values, nil
}

func restoreLocalGitConfigValues(ctx context.Context, root, key string, values []string) {
	if len(values) == 0 {
		_ = BackgroundCommand(ctx, "git", "-C", root, "config", "--local", "--unset-all", key).Run()
		return
	}
	_ = BackgroundCommand(ctx, "git", "-C", root, "config", "--local", "--replace-all", key, values[0]).Run()
	for _, value := range values[1:] {
		_ = BackgroundCommand(ctx, "git", "-C", root, "config", "--local", "--add", key, value).Run()
	}
}

type LegacyPersona struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func PreviewImport(data []byte, host string) ([]Persona, error) {
	var legacy []LegacyPersona
	if err := json.Unmarshal(data, &legacy); err != nil || len(legacy) == 0 {
		return nil, fmt.Errorf("expected a nonempty legacy persona array")
	}
	result := make([]Persona, 0, len(legacy))
	for i, item := range legacy {
		persona := Persona{ID: fmt.Sprintf("imported-%d", i+1), Name: item.Name, GitName: item.Username, GitEmail: item.Email, Host: host, Account: item.Username}
		if err := persona.Validate(); err != nil {
			return nil, fmt.Errorf("legacy persona %d needs a valid name, username, email and GitHub account", i+1)
		}
		result = append(result, persona)
	}
	return result, nil
}
