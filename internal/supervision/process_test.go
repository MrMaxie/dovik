package supervision

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

const processHelperEnvironment = "DOVIK_PROCESS_HELPER"

func TestOwnedProcessWaitReportsExitCode(t *testing.T) {
	command := helperCommand(t, "exit", "7")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	process, err := startOwnedProcess(command)
	if err != nil {
		t.Fatalf("startOwnedProcess() error = %v", err)
	}
	exit := process.Wait()
	if exit.Err != nil {
		t.Fatalf("Wait() error = %v", exit.Err)
	}
	if exit.Code != 7 {
		t.Fatalf("exit code = %d, want 7", exit.Code)
	}
	if stdout.String() != "stdout\n" || stderr.String() != "stderr\n" {
		t.Fatalf("output = (%q, %q), want helper output", stdout.String(), stderr.String())
	}
}

func TestOwnedProcessCanBeStopped(t *testing.T) {
	process, err := startOwnedProcess(helperCommand(t, "wait"))
	if err != nil {
		t.Fatalf("startOwnedProcess() error = %v", err)
	}
	exitChannel := make(chan processExit, 1)
	go func() {
		exitChannel <- process.Wait()
	}()

	if err := process.platform.RequestStop(); err != nil {
		t.Fatalf("RequestStop() error = %v", err)
	}
	select {
	case <-exitChannel:
	case <-time.After(5 * time.Second):
		_ = process.platform.Kill()
		t.Fatal("process did not stop within 5 seconds")
	}
}

func helperCommand(t *testing.T, arguments ...string) *exec.Cmd {
	t.Helper()
	commandArguments := []string{"-test.run=TestProcessHelperProcess", "--"}
	commandArguments = append(commandArguments, arguments...)
	command := exec.Command(os.Args[0], commandArguments...)
	command.Env = append(os.Environ(), processHelperEnvironment+"=1")
	return command
}

func TestProcessHelperProcess(t *testing.T) {
	if os.Getenv(processHelperEnvironment) != "1" {
		return
	}
	separator := 0
	for index, argument := range os.Args {
		if argument == "--" {
			separator = index + 1
			break
		}
	}
	if separator == 0 || separator >= len(os.Args) {
		os.Exit(2)
	}

	switch os.Args[separator] {
	case "exit":
		if len(os.Args) <= separator+1 {
			os.Exit(2)
		}
		code, err := strconv.Atoi(os.Args[separator+1])
		if err != nil {
			os.Exit(2)
		}
		_, _ = os.Stdout.WriteString("stdout\n")
		_, _ = os.Stderr.WriteString("stderr\n")
		os.Exit(code)
	case "wait":
		for {
			time.Sleep(time.Second)
		}
	case "spawn-child":
		child := exec.Command(os.Args[0], "-test.run=TestProcessHelperProcess", "--", "wait")
		child.Env = append(os.Environ(), processHelperEnvironment+"=1")
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		_, _ = os.Stdout.WriteString("child=" + strconv.Itoa(child.Process.Pid) + "\n")
		for {
			time.Sleep(time.Second)
		}
	default:
		os.Exit(2)
	}
}
