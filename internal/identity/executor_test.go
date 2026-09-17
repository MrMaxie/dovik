package identity

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretFilterAcrossWrites(t *testing.T) {
	var output bytes.Buffer
	filter := newSecretFilter(&output, "private-fixture-token")
	for _, part := range []string{"before private-", "fixture-", "token after"} {
		if _, err := filter.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if err := filter.Flush(); err != nil {
		t.Fatal(err)
	}
	if output.String() != "before [credential redacted] after" {
		t.Fatal(output.String())
	}
}

func TestCommandEnvironmentExposesOnlyResolvedGitDirectory(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is unavailable")
	}
	git, err = filepath.Abs(git)
	if err != nil {
		t.Fatal(err)
	}
	git, err = filepath.EvalSymlinks(git)
	if err != nil {
		t.Fatal(err)
	}

	var path string
	for _, entry := range commandEnvironment(t.TempDir(), Persona{Host: "github.com"}, "token") {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, "PATH") {
			path = value
			break
		}
	}
	if path != filepath.Dir(git) {
		t.Fatalf("PATH = %q, want only resolved git directory %q", path, filepath.Dir(git))
	}
}

func TestExecutorUsesPrivateEnvironmentAndPreservesExit(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "fake.go")
	binary := filepath.Join(dir, "original-gh.exe")
	program := `package main
import("fmt";"os";"os/exec")
func main(){ a:=os.Args[1:]; if a[0]=="auth" { fmt.Print("private-fixture-token");return }; if a[0]=="api" { fmt.Print("example");return }; if os.Getenv("GH_TOKEN")!="private-fixture-token" || os.Getenv("GH_DEBUG")!="" || os.Getenv("GITHUB_TOKEN")!="" { os.Exit(90) }; if _,err:=exec.LookPath("git");err!=nil { fmt.Fprint(os.Stderr,err);os.Exit(91) };fmt.Print("out\x00bytes");fmt.Fprint(os.Stderr,"err");os.Exit(7) }`
	if err := os.WriteFile(source, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("go", "build", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, output)
	}
	t.Setenv("GH_DEBUG", "api")
	t.Setenv("GITHUB_TOKEN", "wrong")
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "read-only"}}
	persona := Persona{Host: "github.com", Account: "example"}
	var out, errOut bytes.Buffer
	code, err := (Executor{Directory: dir}).Run(context.Background(), binary, p, persona, Invocation{Arguments: []string{"pr", "view", "1"}}, &out, &errOut)
	if err != nil || code != 7 || out.String() != "out\x00bytes" || errOut.String() != "err" {
		t.Fatalf("code=%d err=%v stdout=%q stderr=%q", code, err, out.String(), errOut.String())
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "gh-call-") {
			t.Fatal("private execution directory leaked")
		}
	}
}

func TestExecutorReturnsSelectedCredentialForGit(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "fake.go")
	binary := filepath.Join(dir, "original-gh.exe")
	program := `package main
import("fmt";"os")
func main(){ a:=os.Args[1:]; if a[0]=="auth" { fmt.Print("private-fixture-token");return }; if a[0]=="api" { fmt.Print("example");return }; os.Exit(90) }`
	if err := os.WriteFile(source, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("go", "build", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, output)
	}
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "maintain"}}
	persona := Persona{Host: "github.com", Account: "example"}
	var out, errOut bytes.Buffer
	code, err := (Executor{Directory: dir}).Run(context.Background(), binary, p, persona, Invocation{
		Arguments: []string{"auth", "git-credential", "get"},
		Input:     []byte("protocol=https\nhost=github.com\npath=owner/repo.git\n\n"),
	}, &out, &errOut)
	if err != nil || code != 0 || errOut.Len() != 0 {
		t.Fatalf("code=%d err=%v stderr=%q", code, err, errOut.String())
	}
	if out.String() != "username=x-access-token\npassword=private-fixture-token\n\n" {
		t.Fatalf("unexpected credential response %q", out.String())
	}
}

func TestExecutorRejectsInteractiveCommandBeforeCredentialAccess(t *testing.T) {
	p := Project{Repository: "owner/repo", Policy: Policy{Preset: "collaborate"}}
	persona := Persona{Host: "github.com", Account: "example"}
	code, err := (Executor{Directory: t.TempDir()}).Run(context.Background(), "missing-gh", p, persona, Invocation{Arguments: []string{"pr", "create"}}, io.Discard, io.Discard)
	if code == 0 || err == nil || !strings.Contains(err.Error(), "operator terminal") {
		t.Fatalf("code=%d err=%v", code, err)
	}
}
