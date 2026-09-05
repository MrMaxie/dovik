//go:build windows

package daemonlaunch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const (
	launcherParentEnvironment = "DOVIK_LAUNCHER_TEST_PARENT"
	launcherChildEnvironment  = "DOVIK_LAUNCHER_TEST_CHILD"
)

func TestDetachedProcessSurvivesLaunchingProcess(t *testing.T) {
	directory := t.TempDir()
	markerPath := filepath.Join(directory, "child-ready")
	pidPath := filepath.Join(directory, "child-pid")
	parent := exec.Command(os.Args[0], "-test.run=TestDetachedLauncherHelper", "--", markerPath, pidPath)
	parent.Env = append(os.Environ(), launcherParentEnvironment+"=1")
	if output, err := parent.CombinedOutput(); err != nil {
		t.Fatalf("launcher helper failed: %v\n%s", err, output)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		marker, markerErr := os.ReadFile(markerPath)
		pidData, pidErr := os.ReadFile(pidPath)
		if markerErr == nil && pidErr == nil {
			if string(marker) != "ready" {
				t.Fatalf("marker = %q", marker)
			}
			pid, err := strconv.Atoi(string(pidData))
			if err != nil {
				t.Fatalf("parse child PID: %v", err)
			}
			process, err := os.FindProcess(pid)
			if err != nil {
				t.Fatalf("find child process: %v", err)
			}
			t.Cleanup(func() { _ = process.Kill() })
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("detached child did not outlive launcher: marker=%v pid=%v", markerErr, pidErr)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestDetachedLauncherHelper(t *testing.T) {
	if os.Getenv(launcherChildEnvironment) == "1" {
		arguments := testHelperArguments()
		if err := os.WriteFile(arguments[0], []byte("ready"), 0o600); err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(arguments[1], []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		return
	}
	if os.Getenv(launcherParentEnvironment) != "1" {
		return
	}
	arguments := testHelperArguments()
	environment := append(os.Environ(), launcherParentEnvironment+"=", launcherChildEnvironment+"=1")
	launcher, err := New(os.Args[0], []string{"-test.run=TestDetachedLauncherHelper", "--", arguments[0], arguments[1]}, environment)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := launcher.Start(); err != nil {
		t.Fatal(err)
	}
}

func testHelperArguments() []string {
	for index, argument := range os.Args {
		if argument == "--" && len(os.Args) == index+3 {
			return os.Args[index+1:]
		}
	}
	fmt.Fprintln(os.Stderr, "missing launcher helper arguments")
	os.Exit(2)
	return nil
}
