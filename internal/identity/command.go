package identity

import (
	"context"
	"os/exec"
)

// BackgroundCommand starts a direct child with piped streams and no Windows console.
func BackgroundCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, name, args...)
	configureBackground(command)
	return command
}
