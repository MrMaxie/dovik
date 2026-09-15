package identity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Executor struct{ Directory string }

func ValidateGH(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("configure an absolute path to the original gh executable")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("original gh executable is unavailable")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("original gh must be a regular executable")
	}
	self, _ := os.Executable()
	selfInfo, _ := os.Stat(self)
	if selfInfo != nil && os.SameFile(info, selfInfo) {
		return "", fmt.Errorf("the gh proxy cannot execute itself")
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(resolved), ".exe") {
		return "", fmt.Errorf("original gh must be an executable, not a shell script")
	}
	probeContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	probe := BackgroundCommand(probeContext, resolved, "--dovik-proxy-identify")
	probe.Env = operatorEnvironment()
	probe.Stderr = io.Discard
	var output limitedBuffer
	output.limit = 1024
	probe.Stdout = &output
	_ = probe.Run()
	if strings.TrimSpace(output.String()) == "dovik-gh-proxy-v1" {
		return "", fmt.Errorf("configure the original GitHub CLI, not the Dovik proxy")
	}
	return resolved, nil
}

func operatorEnvironment() []string {
	keys := []string{"SystemRoot", "WINDIR", "HOME", "USERPROFILE", "LOCALAPPDATA", "APPDATA", "XDG_CONFIG_HOME", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS", "GH_CONFIG_DIR", "TEMP", "TMP", "LANG", "LC_ALL"}
	env := []string{}
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func commandEnvironment(directory string, persona Persona, token string) []string {
	env := []string{"HOME=" + directory, "USERPROFILE=" + directory, "GH_CONFIG_DIR=" + directory, "XDG_CONFIG_HOME=" + directory, "GH_HOST=" + persona.Host, "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "GH_NO_EXTENSION_UPDATE_NOTIFIER=1", "GH_PAGER=cat", "NO_COLOR=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_TERMINAL_PROMPT=0"}
	if runtime.GOOS == "windows" {
		env = append(env, "SystemRoot="+os.Getenv("SystemRoot"), "WINDIR="+os.Getenv("WINDIR"))
	}
	key := "GH_ENTERPRISE_TOKEN"
	if persona.Host == "github.com" || strings.HasSuffix(persona.Host, ".ghe.com") {
		key = "GH_TOKEN"
	}
	return append(env, key+"="+token)
}

func (executor Executor) Run(ctx context.Context, gh string, project Project, persona Persona, in Invocation, stdout, stderr io.Writer) (int, error) {
	command, err := Authorize(project, persona, in)
	if err != nil {
		return 1, err
	}
	if command.Interactive {
		return 1, fmt.Errorf("interactive GitHub CLI commands require the operator terminal")
	}
	gh, err = ValidateGH(gh)
	if err != nil {
		return 1, err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	tokenCommand := BackgroundCommand(ctx, gh, "auth", "token", "--hostname", persona.Host, "--user", persona.Account)
	tokenCommand.Env = operatorEnvironment()
	tokenCommand.Dir = executor.Directory
	var tokenOutput limitedBuffer
	tokenOutput.limit = 64 * 1024
	tokenCommand.Stdout = &tokenOutput
	tokenCommand.Stderr = io.Discard
	tokenCommand.WaitDelay = 2 * time.Second
	if err := tokenCommand.Run(); err != nil {
		return 1, fmt.Errorf("GitHub authentication is unavailable; sign in using the original gh")
	}
	token := strings.TrimSpace(tokenOutput.String())
	if token == "" || strings.ContainsAny(token, "\r\n\x00") {
		return 1, fmt.Errorf("GitHub authentication returned an invalid credential")
	}
	directory, err := os.MkdirTemp(executor.Directory, "gh-call-")
	if err != nil {
		return 1, fmt.Errorf("cannot prepare private gh execution")
	}
	defer os.RemoveAll(directory)
	if err := protectPath(directory); err != nil {
		return 1, err
	}
	env := commandEnvironment(directory, persona, token)
	// Verify the credential's account without exposing authentication output.
	verify := BackgroundCommand(ctx, gh, "api", "user", "--hostname", persona.Host, "--jq", ".login")
	verify.Env = env
	verify.Dir = directory
	verify.Stderr = io.Discard
	verify.WaitDelay = 2 * time.Second
	var login limitedBuffer
	login.limit = 1024
	verify.Stdout = &login
	if err := verify.Run(); err != nil || !strings.EqualFold(strings.TrimSpace(login.String()), persona.Account) {
		return 1, fmt.Errorf("GitHub account verification failed")
	}
	if command.Credential {
		if _, err := fmt.Fprintf(stdout, "username=x-access-token\npassword=%s\n\n", token); err != nil {
			return 1, fmt.Errorf("Git credential output connection closed")
		}
		return 0, nil
	}
	run := BackgroundCommand(ctx, gh, command.Arguments...)
	run.Env = env
	run.Dir = directory
	run.Stdin = bytes.NewReader(in.Input)
	run.WaitDelay = 2 * time.Second
	out := newSecretFilter(stdout, token)
	errOut := newSecretFilter(stderr, token)
	run.Stdout = out
	run.Stderr = errOut
	runErr := run.Run()
	flushErr := errors.Join(out.Flush(), errOut.Flush())
	if flushErr != nil {
		return 1, fmt.Errorf("gh output connection closed")
	}
	if ctx.Err() != nil {
		return 1, fmt.Errorf("gh execution canceled or timed out")
	}
	if runErr == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(runErr, &exit) {
		return exit.ExitCode(), nil
	}
	return 1, fmt.Errorf("original gh could not be executed")
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("private output exceeds limit")
	}
	return b.Buffer.Write(p)
}

// secretFilter retains possible partial matches across process write boundaries.
type secretFilter struct {
	writer  io.Writer
	secret  []byte
	pending []byte
}

func newSecretFilter(w io.Writer, secret string) *secretFilter {
	return &secretFilter{writer: w, secret: []byte(secret)}
}
func (w *secretFilter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
	for len(w.pending) >= len(w.secret) {
		match := bytes.Index(w.pending, w.secret)
		if match >= 0 {
			if match > 0 {
				if _, err := w.writer.Write(w.pending[:match]); err != nil {
					return 0, err
				}
			}
			if _, err := io.WriteString(w.writer, "[credential redacted]"); err != nil {
				return 0, err
			}
			w.pending = w.pending[match+len(w.secret):]
		} else {
			count := len(w.pending) - len(w.secret) + 1
			if _, err := w.writer.Write(w.pending[:count]); err != nil {
				return 0, err
			}
			w.pending = w.pending[count:]
		}
	}
	return len(p), nil
}
func (w *secretFilter) Flush() error {
	_, err := w.writer.Write(w.pending)
	w.pending = nil
	return err
}
