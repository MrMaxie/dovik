package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/MrMaxie/dovik/internal/identityui"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/charmbracelet/x/term"
)

// Run starts the interactive terminal client.
func Run(ctx context.Context, client operatorclient.Client, input io.Reader, output io.Writer) error {
	return RunWithDaemonLauncher(ctx, client, nil, input, output)
}

// RunWithDaemonLauncher starts the interactive terminal client with an optional
// native launcher for an unavailable daemon.
func RunWithDaemonLauncher(ctx context.Context, client operatorclient.Client, launcher DaemonLauncher, input io.Reader, output io.Writer) error {
	defer closeDevLog()
	devLog("run.started")
	inputFile, inputOK := input.(*os.File)
	outputFile, outputOK := output.(*os.File)
	if !inputOK || !outputOK || !term.IsTerminal(inputFile.Fd()) || !term.IsTerminal(outputFile.Fd()) {
		devLog("run.rejected", "reason", "non-interactive terminal")
		return errors.New("tui requires an interactive terminal; use CLI commands when input or output is redirected")
	}
	for {
		program := tea.NewProgram(
			NewModelWithDaemonLauncher(ctx, client, launcher),
			tea.WithContext(ctx),
			tea.WithInput(input),
			tea.WithOutput(output),
		)
		result, err := program.Run()
		if err != nil {
			devLog("run.failed", "error", err.Error())
			return fmt.Errorf("run terminal interface: %w", err)
		}
		model, ok := result.(Model)
		if !ok || !model.configureRequested {
			break
		}
		if err := identityui.Configure(ctx, client, model.configureRoot, input, output); err != nil && !errors.Is(err, huh.ErrUserAborted) {
			return err
		}
	}
	devLog("run.completed")
	return nil
}
