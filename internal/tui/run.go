package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/charmbracelet/x/term"
)

// Run starts the interactive terminal client.
func Run(ctx context.Context, client operatorclient.Client, input io.Reader, output io.Writer) error {
	defer closeDevLog()
	devLog("run.started")
	inputFile, inputOK := input.(*os.File)
	outputFile, outputOK := output.(*os.File)
	if !inputOK || !outputOK || !term.IsTerminal(inputFile.Fd()) || !term.IsTerminal(outputFile.Fd()) {
		devLog("run.rejected", "reason", "non-interactive terminal")
		return errors.New("tui requires an interactive terminal; use CLI commands when input or output is redirected")
	}
	program := tea.NewProgram(
		NewModel(ctx, client),
		tea.WithContext(ctx),
		tea.WithInput(input),
		tea.WithOutput(output),
	)
	if _, err := program.Run(); err != nil {
		devLog("run.failed", "error", err.Error())
		return fmt.Errorf("run terminal interface: %w", err)
	}
	devLog("run.completed")
	return nil
}
