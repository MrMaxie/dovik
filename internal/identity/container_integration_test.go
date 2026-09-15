//go:build integration

package identity

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContainerBridgeIsolation(t *testing.T) {
	for _, backend := range []string{"docker", "podman"} {
		t.Run(backend, func(t *testing.T) {
			if _, err := exec.LookPath(backend); err != nil {
				t.Skip(backend + " unavailable")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			root, err := filepath.Abs(filepath.Join("..", ".."))
			if err != nil {
				t.Fatal(err)
			}
			image := "dovik-identity-integration:dev"
			build := exec.CommandContext(ctx, backend, "build", "--target", "runtime", "--tag", image, root)
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			service := fixtureService(t)
			dir := t.TempDir()
			source := filepath.Join(dir, "fake.go")
			binary := filepath.Join(dir, "original-gh.exe")
			code := `package main
import("fmt";"os")
func main(){ a:=os.Args[1:]; if a[0]=="auth" {fmt.Print("private-fixture-token");return};if a[0]=="api" {fmt.Print("example");return};if os.Getenv("GH_TOKEN")!="private-fixture-token" {os.Exit(90)};fmt.Print("bound-account-result");}`
			if err := os.WriteFile(source, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			if output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, source).CombinedOutput(); err != nil {
				t.Fatalf("%v %s", err, output)
			}
			if _, err := service.Call(ctx, Request{Action: "gh.set", Path: binary}); err != nil {
				t.Fatal(err)
			}
			p := service.Store.Snapshot().Projects[0]
			p.Mode = "agent-isolation"
			if err := service.Store.SetProject(p); err != nil {
				t.Fatal(err)
			}
			var out, errOut bytes.Buffer
			script := `set -eu
test -z "${GH_TOKEN:-}"
test ! -S /var/run/docker.sock
gh pr view 1
if gh auth token; then exit 91; fi
if gh pr view 1 --repo other/repo; then exit 92; fi
if dovik identity list; then exit 93; fi
if env -u DOVIK_AGENT_ENDPOINT dovik identity list; then exit 94; fi
if dovik session create --project project --backend proxy-level; then exit 95; fi
if dovik session persona --persona attacker; then exit 96; fi
test ! -r "$1"
gh pr view 2
`
			exit, err := service.RunContainer(ctx, ContainerRequest{ProjectID: p.ID, Backend: backend, Image: image, Command: []string{"sh", "-c", script, "fixture", service.Store.path}}, &out, &errOut)
			if err != nil || exit != 0 {
				t.Fatalf("exit=%d err=%v\nstdout=%s\nstderr=%s", exit, err, out.String(), errOut.String())
			}
			if strings.Count(out.String(), "bound-account-result") != 2 || strings.Contains(out.String()+errOut.String(), "private-fixture-token") {
				t.Fatalf("unexpected output: %s %s", out.String(), errOut.String())
			}
			if len(service.Snapshot().Sessions) != 0 {
				t.Fatal("session not revoked after agent exit")
			}
		})
	}
}
