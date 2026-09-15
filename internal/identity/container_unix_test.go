//go:build !windows

package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainerConfigurationCanonicalizesProtectedPaths(t *testing.T) {
	realRoot := t.TempDir()
	private := filepath.Join(realRoot, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realRoot, alias); err != nil {
		t.Fatal(err)
	}

	request := ContainerRequest{Backend: "docker", Image: "agent:test", Command: []string{"agent"}}
	if _, err := containerArgs(request, Project{Root: alias}, "test", []string{filepath.Join(alias, "private")}); err == nil {
		t.Fatal("project containing canonicalized private state can be mounted")
	}
}
