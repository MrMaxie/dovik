package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/supervision"
)

type daemonClient interface {
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

// Run executes one CLI command and returns its process exit code.
func Run(ctx context.Context, arguments []string, stdout io.Writer, stderr io.Writer) int {
	endpoint, err := control.DefaultEndpoint()
	if err != nil {
		fmt.Fprintf(stderr, "dovik: %v\n", err)
		return 1
	}
	global := flag.NewFlagSet("dovik", flag.ContinueOnError)
	global.SetOutput(stderr)
	global.StringVar(&endpoint, "endpoint", endpoint, "local daemon endpoint")
	if err := global.Parse(arguments); err != nil {
		return 2
	}
	remaining := global.Args()
	if len(remaining) == 0 {
		writeUsage(stderr)
		return 2
	}
	client := control.NewClient(endpoint)
	if err := runCommand(ctx, client, remaining, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "dovik: %v\n", err)
		return 1
	}
	return 0
}

func runCommand(ctx context.Context, client daemonClient, arguments []string, stdout io.Writer, stderr io.Writer) error {
	switch arguments[0] {
	case "project":
		return runProjectCommand(ctx, client, arguments[1:], stdout, stderr)
	case "process":
		return runProcessCommand(ctx, client, arguments[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func runProjectCommand(ctx context.Context, client daemonClient, arguments []string, stdout io.Writer, stderr io.Writer) error {
	if len(arguments) == 0 {
		return fmt.Errorf("project command is required")
	}
	switch arguments[0] {
	case "add":
		flags := newFlagSet("project add", stderr)
		id := flags.String("id", "", "project ID")
		root := flags.String("root", "", "absolute project root")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		project, err := client.AddProject(ctx, supervision.ProjectDefinition{ID: supervision.ProjectID(*id), RootDirectory: *root})
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "project %s added\n", project.ID)
		return nil
	case "remove":
		flags := newFlagSet("project remove", stderr)
		id := flags.String("id", "", "project ID")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		if err := client.RemoveProject(ctx, supervision.ProjectID(*id)); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "project %s removed\n", *id)
		return nil
	case "list":
		if len(arguments) != 1 {
			return fmt.Errorf("project list accepts no arguments")
		}
		projects, err := client.ListProjects(ctx)
		if err != nil {
			return err
		}
		for _, project := range projects {
			fmt.Fprintf(stdout, "%s\t%s\n", project.ID, project.RootDirectory)
		}
		return nil
	default:
		return fmt.Errorf("unknown project command %q", arguments[0])
	}
}

func runProcessCommand(ctx context.Context, client daemonClient, arguments []string, stdout io.Writer, stderr io.Writer) error {
	if len(arguments) == 0 {
		return fmt.Errorf("process command is required")
	}
	switch arguments[0] {
	case "add":
		flags := newFlagSet("process add", stderr)
		projectID := flags.String("project", "", "project ID")
		processID := flags.String("id", "", "process ID")
		command := flags.String("command", "", "executable command")
		workingDirectory := flags.String("working-directory", "", "working directory")
		var commandArguments stringList
		var environment stringList
		flags.Var(&commandArguments, "arg", "command argument (repeatable)")
		flags.Var(&environment, "env", "environment override KEY=VALUE (repeatable)")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		overrides, err := parseEnvironment(environment)
		if err != nil {
			return err
		}
		process, err := client.AddProcess(ctx, supervision.ProcessDefinition{
			ProjectID:            supervision.ProjectID(*projectID),
			ID:                   supervision.ProcessID(*processID),
			Command:              *command,
			Arguments:            commandArguments,
			WorkingDirectory:     *workingDirectory,
			EnvironmentOverrides: overrides,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "process %s/%s added\n", process.ProjectID, process.ID)
		return nil
	case "remove":
		projectID, processID, err := parseProcessReference("process remove", arguments[1:], stderr)
		if err != nil {
			return err
		}
		if err := client.RemoveProcess(ctx, projectID, processID); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "process %s/%s removed\n", projectID, processID)
		return nil
	case "list":
		flags := newFlagSet("process list", stderr)
		projectID := flags.String("project", "", "project ID")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		processes, err := client.ListProcesses(ctx, supervision.ProjectID(*projectID))
		if err != nil {
			return err
		}
		for _, process := range processes {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", process.ID, process.Command, strings.Join(process.Arguments, " "))
		}
		return nil
	case "start", "stop", "restart":
		projectID, processID, err := parseProcessReference("process "+arguments[0], arguments[1:], stderr)
		if err != nil {
			return err
		}
		var runtimeState supervision.ProcessRuntime
		switch arguments[0] {
		case "start":
			runtimeState, err = client.Start(ctx, projectID, processID)
		case "stop":
			runtimeState, err = client.Stop(ctx, projectID, processID)
		case "restart":
			runtimeState, err = client.Restart(ctx, projectID, processID)
		}
		if err != nil {
			return err
		}
		writeRuntime(stdout, runtimeState)
		return nil
	case "status":
		projectID, processID, err := parseProcessReference("process status", arguments[1:], stderr)
		if err != nil {
			return err
		}
		runtimeState, exists, err := client.Status(ctx, projectID, processID)
		if err != nil {
			return err
		}
		if !exists {
			fmt.Fprintf(stdout, "%s/%s\t%s\n", projectID, processID, supervision.ProcessStateStopped)
			return nil
		}
		writeRuntime(stdout, runtimeState)
		return nil
	case "logs":
		flags := newFlagSet("process logs", stderr)
		projectID := flags.String("project", "", "project ID")
		processID := flags.String("process", "", "process ID")
		tailLimit := flags.Int("tail", 100, "maximum retained events")
		after := flags.Uint64("after", 0, "return events after this sequence")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		tail, err := client.Logs(ctx, supervision.ProjectID(*projectID), supervision.ProcessID(*processID), *after, *tailLimit)
		if err != nil {
			return err
		}
		if tail.Truncated {
			fmt.Fprintln(stdout, "[output truncated]")
		}
		for _, event := range tail.Events {
			fmt.Fprintf(stdout, "[%s] %s", event.Stream, event.Data)
			if len(event.Data) == 0 || event.Data[len(event.Data)-1] != '\n' {
				fmt.Fprintln(stdout)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown process command %q", arguments[0])
	}
}

func parseProcessReference(name string, arguments []string, stderr io.Writer) (supervision.ProjectID, supervision.ProcessID, error) {
	flags := newFlagSet(name, stderr)
	projectID := flags.String("project", "", "project ID")
	processID := flags.String("process", "", "process ID")
	if err := parseExact(flags, arguments); err != nil {
		return "", "", err
	}
	return supervision.ProjectID(*projectID), supervision.ProcessID(*processID), nil
}

func parseExact(flags *flag.FlagSet, arguments []string) error {
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	return nil
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	return flags
}

func parseEnvironment(values []string) (map[string]string, error) {
	overrides := make(map[string]string, len(values))
	for _, value := range values {
		key, content, found := strings.Cut(value, "=")
		if !found || key == "" {
			return nil, fmt.Errorf("invalid environment override %q, want KEY=VALUE", value)
		}
		overrides[key] = content
	}
	return overrides, nil
}

func writeRuntime(writer io.Writer, runtimeState supervision.ProcessRuntime) {
	pid := "-"
	if runtimeState.PID != nil {
		pid = strconv.Itoa(*runtimeState.PID)
	}
	fmt.Fprintf(writer, "%s/%s\t%s\tpid=%s\tinstance=%s\n", runtimeState.ProjectID, runtimeState.ProcessID, runtimeState.State, pid, runtimeState.InstanceID)
}

func writeUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: dovik [--endpoint PATH] project|process COMMAND [OPTIONS]")
}

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}
