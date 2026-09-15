package identity

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestBackgroundChildHasNoConsoleAndPreservesStreams(t *testing.T) {
	if os.Getenv("DOVIK_NO_CONSOLE_FIXTURE") == "1" {
		kernel := windows.NewLazySystemDLL("kernel32.dll")
		window, _, _ := kernel.NewProc("GetConsoleWindow").Call()
		if window != 0 {
			fmt.Fprintf(os.Stderr, "console window=%d", window)
			os.Exit(91)
		}
		if os.Getenv("DOVIK_NESTED_FIXTURE") != "1" {
			binary, _ := os.Executable()
			child := exec.Command(binary, "-test.run=^TestBackgroundChildHasNoConsoleAndPreservesStreams$")
			child.Env = append(os.Environ(), "DOVIK_NESTED_FIXTURE=1")
			child.Stdout, child.Stderr = os.Stdout, os.Stderr
			if err := child.Run(); err != nil {
				os.Exit(92)
			}
		}
		fmt.Fprint(os.Stdout, "stdout fixture")
		fmt.Fprint(os.Stderr, "stderr fixture")
		os.Exit(0)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := BackgroundCommand(context.Background(), binary, "-test.run=^TestBackgroundChildHasNoConsoleAndPreservesStreams$")
	command.Env = append(os.Environ(), "DOVIK_NO_CONSOLE_FIXTURE=1")
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "stdout fixture") || !strings.Contains(string(output), "stderr fixture") {
		t.Fatalf("child console/streams: %v %q", err, output)
	}
}
