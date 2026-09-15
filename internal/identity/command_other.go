//go:build !windows

package identity

import "os/exec"

func configureBackground(command *exec.Cmd) {}
