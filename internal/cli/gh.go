package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/MrMaxie/dovik/internal/identity"
)

type ghClient interface {
	Identity(context.Context, identity.Request) (identity.Snapshot, error)
	ExecuteGH(context.Context, identity.ExecutionRequest, io.Writer, io.Writer) (int, error)
}

type ghRootResolver func(context.Context, string) (string, error)
type ghOriginalRunner func(context.Context, string, []string, io.Reader, io.Writer, io.Writer, []string) (int, error)

func runGH(ctx context.Context, client ghClient, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return runGHWith(ctx, client, args, stdin, stdout, stderr, identity.GitRoot, runOriginalGH)
}

func runGHWith(ctx context.Context, client ghClient, args []string, stdin io.Reader, stdout, stderr io.Writer, resolveRoot ghRootResolver, runOriginal ghOriginalRunner) (int, error) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 1 && args[0] == "--dovik-proxy-identify" {
		_, err := fmt.Fprintln(stdout, "dovik-gh-proxy-v1")
		return 0, err
	}
	if endpoint := os.Getenv("DOVIK_AGENT_ENDPOINT"); endpoint != "" {
		in, err := prepareInvocation(args, stdin)
		if err != nil {
			return 2, err
		}
		connection, err := identity.DialAgent(ctx, endpoint)
		if err != nil {
			return 1, fmt.Errorf("isolated agent channel is unavailable")
		}
		defer connection.Close()
		finished := make(chan struct{})
		defer close(finished)
		go func() {
			select {
			case <-ctx.Done():
				connection.Close()
			case <-finished:
			}
		}()
		if err := json.NewEncoder(connection).Encode(identity.AgentRequest{Version: 1, Action: "execute", Invocation: in}); err != nil {
			return 1, err
		}
		return identity.ReadStream(connection, stdout, stderr)
	}
	if ordinaryGHMetadata(args) {
		return runOriginal(ctx, "", args, stdin, stdout, stderr, nil)
	}
	snapshot, err := client.Identity(ctx, identity.Request{Action: "get"})
	if err != nil {
		return runOriginal(ctx, "", args, stdin, stdout, stderr, nil)
	}
	root, err := resolveRoot(ctx, ".")
	if err != nil {
		return runOriginal(ctx, snapshot.State.GHPath, args, stdin, stdout, stderr, nil)
	}
	project, err := snapshot.State.FindProject("", root)
	if err != nil {
		return runOriginal(ctx, snapshot.State.GHPath, args, stdin, stdout, stderr, nil)
	}
	if !project.ProxyEnabled && project.Mode == "proxy-level" && os.Getenv("DOVIK_SESSION") == "" {
		return runOriginal(ctx, snapshot.State.GHPath, args, stdin, stdout, stderr, nil)
	}
	if gitCredentialNotification(args) {
		count, err := io.Copy(io.Discard, io.LimitReader(stdin, 64*1024+1))
		if err != nil || count > 64*1024 {
			return 1, fmt.Errorf("Git credential notification cannot be read")
		}
		return 0, nil
	}
	if len(args) == 2 && args[0] == "pr" && args[1] == "create" {
		persona, err := snapshot.State.FindPersona(project.Persona)
		if err != nil {
			return 1, err
		}
		command, authorizationErr := identity.Authorize(project, persona, identity.Invocation{Arguments: args})
		if authorizationErr == nil && command.Interactive {
			return runInteractiveGH(ctx, client, snapshot.State.GHPath, project, persona, command.Arguments, stdin, stdout, stderr, runOriginal)
		}
	}
	in, err := prepareInvocation(args, stdin)
	if err != nil {
		return 2, err
	}
	return client.ExecuteGH(ctx, identity.ExecutionRequest{ProjectID: project.ID, SessionID: os.Getenv("DOVIK_SESSION"), Invocation: in}, stdout, stderr)
}

func runInteractiveGH(ctx context.Context, client ghClient, path string, project identity.Project, persona identity.Persona, args []string, stdin io.Reader, stdout, stderr io.Writer, runOriginal ghOriginalRunner) (int, error) {
	request := identity.ExecutionRequest{ProjectID: project.ID, Invocation: identity.Invocation{
		Arguments: []string{"auth", "git-credential", "get"},
		Input:     []byte("protocol=https\nhost=" + persona.Host + "\npath=" + project.Repository + "\n\n"),
	}}
	var credential bytes.Buffer
	code, err := client.ExecuteGH(ctx, request, &credential, io.Discard)
	if err != nil {
		return 1, err
	}
	if code != 0 {
		return code, nil
	}
	token := ""
	for _, line := range strings.Split(strings.ReplaceAll(credential.String(), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "password=") {
			token = strings.TrimPrefix(line, "password=")
			break
		}
	}
	if token == "" || strings.ContainsAny(token, "\r\n\x00") {
		return 1, errors.New("GitHub authentication is unavailable")
	}
	return runOriginal(ctx, path, args, stdin, stdout, stderr, interactiveGHEnvironment(persona, token))
}

func interactiveGHEnvironment(persona identity.Persona, token string) []string {
	blocked := map[string]bool{
		"GH_TOKEN": true, "GITHUB_TOKEN": true, "GH_ENTERPRISE_TOKEN": true,
		"GITHUB_ENTERPRISE_TOKEN": true, "GH_HOST": true, "GH_PROMPT_DISABLED": true,
	}
	environment := []string{}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if !blocked[strings.ToUpper(key)] {
			environment = append(environment, item)
		}
	}
	tokenKey := "GH_ENTERPRISE_TOKEN"
	if persona.Host == "github.com" || strings.HasSuffix(persona.Host, ".ghe.com") {
		tokenKey = "GH_TOKEN"
	}
	return append(environment, "GH_HOST="+persona.Host, tokenKey+"="+token)
}

func gitCredentialNotification(args []string) bool {
	return len(args) == 3 && args[0] == "auth" && args[1] == "git-credential" && (args[2] == "store" || args[2] == "erase")
}

func ordinaryGHMetadata(args []string) bool {
	if len(args) == 0 {
		return true
	}
	if args[0] == "help" || args[0] == "--version" || args[0] == "version" {
		return true
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func runOriginalGH(ctx context.Context, path string, args []string, stdin io.Reader, stdout, stderr io.Writer, environment []string) (int, error) {
	if path == "" {
		var err error
		path, err = exec.LookPath("realgh")
		if err != nil {
			return 1, errors.New("GitHub CLI is unavailable")
		}
	}
	path, err := identity.ValidateGH(path)
	if err != nil {
		return 1, errors.New("GitHub CLI is unavailable")
	}
	command := exec.CommandContext(ctx, path, args...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if environment != nil {
		command.Env = environment
	}
	if err := command.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 1, errors.New("GitHub CLI is unavailable")
	}
	return 0, nil
}
