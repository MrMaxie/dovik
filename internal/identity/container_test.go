package identity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerConfigurationRejectsCredentialMounts(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	r := ContainerRequest{Backend: "docker", Image: "agent:test", Command: []string{"agent"}}
	if _, err := containerArgs(r, Project{Root: root}, "test", []string{private}); err == nil {
		t.Fatal("parent of private state can be mounted")
	}
	project := t.TempDir()
	for _, backend := range []string{"docker", "podman"} {
		r.Backend = backend
		args, err := containerArgs(r, Project{Root: project}, "test", []string{private})
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(args, " ")
		for _, flag := range []string{"--cap-drop=ALL", "--security-opt=no-new-privileges", "--user 10001:10001", "--pull=never"} {
			if !strings.Contains(joined, flag) {
				t.Fatal(args)
			}
		}
	}
	r.Resources = []Resource{{Source: private, Target: "/resources/private"}}
	if _, err := containerArgs(r, Project{Root: project}, "test", []string{private}); err == nil {
		t.Fatal("private resource accepted")
	}
}
