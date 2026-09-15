package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestReleaseWorkflowsAreValidYAML(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) < 4 {
		t.Fatalf("workflow count = %d, want at least 4", len(workflows))
	}
	for _, path := range workflows {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := yaml.Unmarshal(content, &document); err != nil {
				t.Fatal(err)
			}
			if document["name"] == nil || document["on"] == nil || document["jobs"] == nil {
				t.Fatal("workflow requires name, on, and jobs")
			}
		})
	}
}

func TestSafeExtractPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := safeExtractPath(root, "../escape"); err == nil {
		t.Fatal("traversal path accepted")
	}
	path, err := safeExtractPath(root, "dovik/file")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, root) {
		t.Fatalf("path = %q", path)
	}
}

func TestCurrentTargetIsSupportedOnReleaseHosts(t *testing.T) {
	_, err := currentTarget()
	if (runtime.GOOS == "windows" || runtime.GOOS == "linux" || runtime.GOOS == "darwin") && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") && err != nil {
		t.Fatal(err)
	}
}
