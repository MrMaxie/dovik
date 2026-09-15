package identity

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Resource struct {
	Source string `json:"source"`
	Target string `json:"target"`
}
type ContainerRequest struct {
	ProjectID string     `json:"projectId"`
	Backend   string     `json:"backend"`
	Image     string     `json:"image"`
	Command   []string   `json:"command"`
	Resources []Resource `json:"resources,omitempty"`
	Input     []byte     `json:"input,omitempty"`
}

func within(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func validateMount(source string, protected []string) (string, error) {
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil || !filepath.IsAbs(resolved) || strings.ContainsAny(resolved, ",\r\n") {
		return "", fmt.Errorf("resource must be an existing absolute path without commas")
	}
	for _, path := range protected {
		if path == "" {
			continue
		}
		absolute, e := filepath.Abs(path)
		if e != nil {
			continue
		}
		if canonical, evalErr := filepath.EvalSymlinks(absolute); evalErr == nil {
			absolute = canonical
		}
		if within(absolute, resolved) || within(resolved, absolute) {
			return "", fmt.Errorf("resource overlaps operator credentials, executables, or control state")
		}
	}
	// A read-only bind of a socket still permits engine API calls. Reject special
	// files anywhere in mounted trees, including nested engine/control sockets.
	err = filepath.WalkDir(resolved, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("cannot inspect resource contents")
		}
		if entry.Type()&(os.ModeSocket|os.ModeDevice|os.ModeNamedPipe) != 0 {
			return fmt.Errorf("resources must not contain sockets, devices, or named pipes")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func containerArgs(r ContainerRequest, p Project, name string, protected []string) ([]string, error) {
	if r.Backend != "docker" && r.Backend != "podman" {
		return nil, fmt.Errorf("select docker or podman")
	}
	if r.Image == "" || strings.HasPrefix(r.Image, "-") || strings.ContainsAny(r.Image, " \r\n\x00") || len(r.Command) == 0 || len(r.Input) > 1024*1024 {
		return nil, fmt.Errorf("a valid image and agent command are required")
	}
	root, err := validateMount(p.Root, protected)
	if err != nil {
		return nil, err
	}
	args := []string{"run", "--detach", "--pull=never", "--name", name, "--user", "10001:10001", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=512", "--memory=4g", "--cpus=2", "--workdir", "/workspace", "--env", "DOVIK_AGENT_ENDPOINT=/tmp/dovik-agent.sock", "--mount", "type=bind,src=" + root + ",dst=/workspace", "--entrypoint", "/usr/local/bin/dovik"}
	for _, resource := range r.Resources {
		source, err := validateMount(resource.Source, protected)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(resource.Target, "/resources/") || strings.ContainsAny(resource.Target, ",\r\n") || strings.Contains(resource.Target, "..") {
			return nil, fmt.Errorf("additional read-only resources must be mounted under /resources/")
		}
		args = append(args, "--mount", "type=bind,readonly,src="+source+",dst="+resource.Target)
	}
	return append(args, r.Image, "agent-idle"), nil
}

func (s *Service) RunContainer(ctx context.Context, r ContainerRequest, stdout, stderr io.Writer) (int, error) {
	state := s.Store.Snapshot()
	p, err := state.FindProject(r.ProjectID, "")
	if err != nil {
		return 1, err
	}
	runtimePath, err := exec.LookPath(r.Backend)
	if err != nil {
		return 1, fmt.Errorf("selected container runtime is unavailable")
	}
	home, _ := os.UserHomeDir()
	config, _ := os.UserConfigDir()
	executable, _ := os.Executable()
	protected := []string{filepath.Dir(s.Store.path), executable, state.GHPath, filepath.Join(home, ".ssh"), filepath.Join(home, ".docker"), filepath.Join(home, ".config", "containers"), filepath.Join(home, ".config", "gh"), filepath.Join(config, "gh"), os.Getenv("GH_CONFIG_DIR"), "/var/run/docker.sock", "/run/podman", filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "podman")}
	session, err := s.createSession(Request{ProjectID: p.ID, Backend: r.Backend})
	if err != nil {
		return 1, err
	}
	defer s.Call(context.Background(), Request{Action: "session.revoke", SessionID: session.ID})
	name := "dovik-agent-" + session.ID[:16]
	args, err := containerArgs(r, p, name, protected)
	if err != nil {
		return 1, err
	}
	runContext, cancel := context.WithCancel(ctx)
	defer cancel()
	create := BackgroundCommand(runContext, runtimePath, args...)
	create.Stdout = io.Discard
	create.Stderr = stderr
	if err := create.Run(); err != nil {
		return 1, fmt.Errorf("container could not start; prepare the selected image and runtime")
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_ = BackgroundCommand(cleanupCtx, runtimePath, "rm", "--force", name).Run()
	}()
	bridge := BackgroundCommand(runContext, runtimePath, "exec", "-i", name, "/usr/local/bin/dovik", "agent-bridge")
	bridge.WaitDelay = 2 * time.Second
	incoming, err := bridge.StdoutPipe()
	if err != nil {
		return 1, err
	}
	outgoing, err := bridge.StdinPipe()
	if err != nil {
		return 1, err
	}
	bridge.Stderr = stderr
	if err := bridge.Start(); err != nil {
		return 1, fmt.Errorf("container bridge could not start")
	}
	defer func() { cancel(); outgoing.Close(); _ = bridge.Wait() }()
	reader := bufio.NewReader(incoming)
	ready := make(chan error, 1)
	go func() {
		line, err := reader.ReadSlice('\n')
		if err == nil && string(line) != "dovik-agent-ready-v1\n" {
			err = fmt.Errorf("incompatible agent bridge")
		}
		ready <- err
	}()
	select {
	case err := <-ready:
		if err != nil {
			return 1, fmt.Errorf("container image does not provide a compatible Dovik bridge")
		}
	case <-time.After(15 * time.Second):
		return 1, fmt.Errorf("container bridge readiness timed out")
	case <-ctx.Done():
		return 1, ctx.Err()
	}
	var bridgeMu sync.Mutex
	s.activateSession(session.ID)
	go func() {
		for {
			if _, err := reader.Peek(1); err != nil {
				cancel()
				return
			}
			bridgeMu.Lock()
			s.HandleAgent(runContext, session.ID, readWriter{reader, outgoing})
			bridgeMu.Unlock()
		}
	}()
	agentArgs := append([]string{"exec", "-i", name}, r.Command...)
	agent := BackgroundCommand(runContext, runtimePath, agentArgs...)
	agent.Stdin = strings.NewReader(string(r.Input))
	agent.Stdout = stdout
	agent.Stderr = stderr
	agent.WaitDelay = 2 * time.Second
	if err := agent.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 1, fmt.Errorf("agent command could not start")
	}
	return 0, nil
}
