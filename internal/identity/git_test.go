package identity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitAuthorOnlyChangesSelectedRepository(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(t.TempDir(), "gitconfig")
	initial := []byte("[user]\n name = Global\n email = global@example.test\n")
	if err := os.WriteFile(global, initial, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	p := Persona{ID: "work", Name: "Work", GitName: "Local Author", GitEmail: "local@example.test", Host: "github.com", Account: "example"}
	if err := ApplyGitAuthor(context.Background(), root, p); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"user.name": p.GitName, "user.email": p.GitEmail} {
		output, err := exec.Command("git", "-C", root, "config", "--local", "--get", key).Output()
		if err != nil || strings.TrimSpace(string(output)) != want {
			t.Fatal("incorrect local author", key, err)
		}
	}
	after, _ := os.ReadFile(global)
	if string(after) != string(initial) {
		t.Fatal("global author changed")
	}
}

func TestGitCredentialHelperRoutesOnlySelectedRepositoryThroughGH(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(t.TempDir(), "gitconfig")
	initial := []byte("[credential]\n helper = store\n")
	if err := os.WriteFile(global, initial, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if err := ApplyGitCredentialHelper(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("git", "-C", root, "config", "--local", "--null", "--get-all", "credential.helper").Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "\x00!gh auth git-credential\x00" {
		t.Fatalf("unexpected local helpers %q", output)
	}
	after, _ := os.ReadFile(global)
	if string(after) != string(initial) {
		t.Fatal("global credential helper changed")
	}
}

func TestImportPreviewAndAtomicCollision(t *testing.T) {
	personas, err := PreviewImport([]byte(`[{"name":"Work","username":"example","email":"example@example.test"}]`), "github.com")
	if err != nil || len(personas) != 1 {
		t.Fatal(err)
	}
	s := fixtureService(t)
	before := s.Snapshot()
	if len(before.State.Personas) != 1 {
		t.Fatal("preview mutated state")
	}
	if _, err := s.Call(context.Background(), Request{Action: "persona.import", Personas: personas}); err != nil {
		t.Fatal(err)
	}
	duplicate := append([]Persona{{ID: "new", Name: "New", GitName: "New", GitEmail: "new@example.test", Host: "github.com", Account: "new"}}, personas...)
	if _, err := s.Call(context.Background(), Request{Action: "persona.import", Personas: duplicate}); err == nil {
		t.Fatal("import overwrote an existing persona")
	}
	if len(s.Snapshot().State.Personas) != 2 {
		t.Fatal("failed import was partially saved")
	}
}
