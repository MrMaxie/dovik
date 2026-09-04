// Package operatorclient defines the daemon operations shared by operator
// surfaces. Implementations must delegate lifecycle authority to the daemon.
package operatorclient

import (
	"context"

	"github.com/MrMaxie/dovik/internal/supervision"
)

// Client is the complete local control-plane surface used by the CLI and TUI.
type Client interface {
	AddProject(context.Context, supervision.ProjectDefinition) (supervision.ProjectDefinition, error)
	RemoveProject(context.Context, supervision.ProjectID) error
	ListProjects(context.Context) ([]supervision.ProjectDefinition, error)
	AddProcess(context.Context, supervision.ProcessDefinition) (supervision.ProcessDefinition, error)
	RemoveProcess(context.Context, supervision.ProjectID, supervision.ProcessID) error
	ListProcesses(context.Context, supervision.ProjectID) ([]supervision.ProcessDefinition, error)
	Start(context.Context, supervision.ProjectID, supervision.ProcessID) (supervision.ProcessRuntime, error)
	Stop(context.Context, supervision.ProjectID, supervision.ProcessID) (supervision.ProcessRuntime, error)
	Restart(context.Context, supervision.ProjectID, supervision.ProcessID) (supervision.ProcessRuntime, error)
	Status(context.Context, supervision.ProjectID, supervision.ProcessID) (supervision.ProcessRuntime, bool, error)
	Logs(context.Context, supervision.ProjectID, supervision.ProcessID, uint64, int) (supervision.OutputTail, error)
}
