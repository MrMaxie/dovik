package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionDoesNotStartDaemon(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := runCLI([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != "dovikd 1.0.0\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestHelpDoesNotStartDaemon(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := runCLI([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage: dovikd") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUnknownArgumentFailsWithoutStartingDaemon(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := runCLI([]string{"--unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown argument") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
