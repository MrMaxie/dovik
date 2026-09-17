package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MrMaxie/dovik/internal/buildinfo"
	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/mcpoperator"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/MrMaxie/dovik/internal/supervision"
	"github.com/MrMaxie/dovik/internal/terminalstyle"
	"github.com/charmbracelet/x/term"
)

type outputMode struct {
	json bool
}

type commandError struct {
	code    string
	message string
	cause   error
}

func (err *commandError) Error() string { return err.message }
func (err *commandError) Unwrap() error { return err.cause }

// Run executes one CLI command and returns its process exit code.
func Run(ctx context.Context, arguments []string, stdout io.Writer, stderr io.Writer) int {
	return RunIO(ctx, arguments, os.Stdin, stdout, stderr)
}

// RunIO executes one command with explicit terminal streams.
func RunIO(ctx context.Context, arguments []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(arguments) == 1 && arguments[0] == "--version" {
		fmt.Fprintf(stdout, "dovik %s\n", buildinfo.Version)
		return 0
	}
	if topic, ok := helpTopic(arguments); ok {
		writeHelp(stdout, topic)
		return 0
	}
	if len(arguments) == 1 && arguments[0] == "agent-idle" {
		<-ctx.Done()
		return 0
	}
	if len(arguments) == 1 && arguments[0] == "agent-bridge" {
		if err := identity.RunBridge(ctx, stdin, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	wantsJSON := hasJSONFlag(arguments)
	defaultEndpoint, err := control.DefaultEndpoint()
	if err != nil {
		return writeFailure(stderr, wantsJSON, &commandError{code: "configuration_error", message: err.Error(), cause: err}, 1)
	}
	endpoint := defaultEndpoint

	global := flag.NewFlagSet("dovik", flag.ContinueOnError)
	if wantsJSON {
		global.SetOutput(io.Discard)
	} else {
		global.SetOutput(stderr)
	}
	global.StringVar(&endpoint, "endpoint", endpoint, "local daemon endpoint")
	jsonOutput := global.Bool("json", false, "emit one JSON document")
	if err := global.Parse(arguments); err != nil {
		return writeFailure(stderr, wantsJSON, invalidArguments(err), 2)
	}
	mode := outputMode{json: *jsonOutput}
	remaining := global.Args()
	if len(remaining) == 0 {
		if mode.json {
			return writeFailure(stderr, true, invalidArguments(errors.New("command is required")), 2)
		}
		if !interactive(stdin, stdout) || !rootEntryAllowed() {
			writeUsage(stdout)
			return 0
		}
		client := control.NewClient(endpoint)
		if err := runRootEntry(ctx, client, endpoint, defaultEndpoint, stdin, stdout); err != nil {
			return writeFailure(stderr, false, classifyClientError(err), 1)
		}
		return 0
	}

	client := control.NewClient(endpoint)
	if len(remaining) > 1 && remaining[0] == "session" && remaining[1] == "run" {
		code, err := runContainerSession(ctx, client, remaining[2:], stdin, stdout, stderr, mode)
		if err != nil {
			return writeFailure(stderr, mode.json, &commandError{code: "session_error", message: err.Error()}, 1)
		}
		return code
	}
	if remaining[0] == "whoami" {
		if len(remaining) != 1 {
			return writeFailure(stderr, mode.json, invalidArguments(errors.New("whoami accepts no arguments")), 2)
		}
		if err := runWhoAmI(ctx, client, stdout, mode); err != nil {
			return writeFailure(stderr, mode.json, classifyClientError(err), 1)
		}
		return 0
	}
	if os.Getenv("DOVIK_AGENT_ENDPOINT") != "" && remaining[0] != "gh" && remaining[0] != "session" {
		return writeFailure(stderr, mode.json, &commandError{code: "permission_denied", message: "use the operator terminal to administer Dovik"}, 1)
	}
	if remaining[0] == "gh" {
		if mode.json {
			return writeFailure(stderr, true, invalidArguments(fmt.Errorf("place gh output flags after gh --")), 2)
		}
		code, err := runGH(ctx, client, remaining[1:], stdin, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "gh: %s\n", err)
			return code
		}
		return code
	}
	if remaining[0] == "identity" || remaining[0] == "policy" || remaining[0] == "session" || remaining[0] == "doctor" || (remaining[0] == "project" && len(remaining) > 1 && remaining[1] == "configure") {
		var err error
		switch remaining[0] {
		case "identity":
			err = runIdentityCommand(ctx, client, remaining, stdin, stdout, stderr, mode)
		case "policy":
			err = runPolicyCommand(ctx, client, remaining, stdout, stderr, mode)
		case "session":
			err = runSessionCommand(ctx, client, remaining, stdout, stderr, mode)
		case "doctor":
			err = runDoctor(ctx, client, remaining[1:], stdout, stderr, mode)
		case "project":
			err = runProjectConfigure(ctx, client, remaining[2:], stdin, stdout, stderr, mode)
		}
		if err != nil {
			return writeFailure(stderr, mode.json, &commandError{code: "identity_error", message: err.Error()}, 1)
		}
		return 0
	}
	if remaining[0] == "mcp" {
		if mode.json || len(remaining) != 1 {
			return writeFailure(stderr, mode.json, invalidArguments(errors.New("mcp accepts no arguments and does not support --json")), 2)
		}
		reader, ok := stdin.(io.ReadCloser)
		if !ok {
			reader = io.NopCloser(stdin)
		}
		if err := mcpoperator.Run(ctx, client, reader, stdout, stderr); err != nil {
			if ctx.Err() != nil {
				return 0
			}
			return writeFailure(stderr, false, &commandError{code: "mcp_error", message: "MCP session failed.", cause: err}, 1)
		}
		return 0
	}
	if remaining[0] == "tui" {
		if mode.json {
			return writeFailure(stderr, true, invalidArguments(errors.New("tui does not support --json")), 2)
		}
		if len(remaining) != 1 {
			return writeFailure(stderr, false, invalidArguments(errors.New("tui accepts no arguments")), 2)
		}
		if err := runTUI(ctx, client, endpoint, defaultEndpoint, stdin, stdout); err != nil {
			return writeFailure(stderr, false, classifyClientError(err), 1)
		}
		return 0
	}

	commandContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := runCommand(commandContext, client, remaining, stdout, stderr, mode); err != nil {
		classified := classifyClientError(err)
		exitCode := 1
		if classified.code == "invalid_arguments" {
			exitCode = 2
		}
		return writeFailure(stderr, mode.json, classified, exitCode)
	}
	return 0
}

func runCommand(ctx context.Context, client operatorclient.Client, arguments []string, stdout io.Writer, stderr io.Writer, mode outputMode) error {
	switch arguments[0] {
	case "project":
		return runProjectCommand(ctx, client, arguments[1:], stdout, stderr, mode)
	case "process":
		return runProcessCommand(ctx, client, arguments[1:], stdout, stderr, mode)
	default:
		return invalidArguments(fmt.Errorf("unknown command %q", arguments[0]))
	}
}

func runProjectCommand(ctx context.Context, client operatorclient.Client, arguments []string, stdout io.Writer, stderr io.Writer, mode outputMode) error {
	if len(arguments) == 0 {
		return invalidArguments(errors.New("project command is required"))
	}
	switch arguments[0] {
	case "add":
		flags := newFlagSet("project add", stderr, mode)
		id := flags.String("id", "", "project ID")
		root := flags.String("root", "", "absolute project root")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		project, err := client.AddProject(ctx, supervision.ProjectDefinition{ID: supervision.ProjectID(*id), RootDirectory: *root})
		if err != nil {
			return err
		}
		if mode.json {
			return writeJSON(stdout, projectResult(project))
		}
		fmt.Fprintf(stdout, "project %s added\n", project.ID)
		return nil
	case "remove":
		flags := newFlagSet("project remove", stderr, mode)
		id := flags.String("id", "", "project ID")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		if err := client.RemoveProject(ctx, supervision.ProjectID(*id)); err != nil {
			return err
		}
		if mode.json {
			return writeJSON(stdout, removalResult{ProjectID: *id, Removed: true})
		}
		fmt.Fprintf(stdout, "project %s removed\n", *id)
		return nil
	case "list":
		if len(arguments) != 1 {
			return invalidArguments(errors.New("project list accepts no arguments"))
		}
		projects, err := client.ListProjects(ctx)
		if err != nil {
			return err
		}
		if mode.json {
			results := make([]projectJSON, 0, len(projects))
			for _, project := range projects {
				results = append(results, projectResult(project))
			}
			return writeJSON(stdout, results)
		}
		for _, project := range projects {
			fmt.Fprintf(stdout, "%s\t%s\n", project.ID, project.RootDirectory)
		}
		return nil
	default:
		return invalidArguments(fmt.Errorf("unknown project command %q", arguments[0]))
	}
}

func runProcessCommand(ctx context.Context, client operatorclient.Client, arguments []string, stdout io.Writer, stderr io.Writer, mode outputMode) error {
	if len(arguments) == 0 {
		return invalidArguments(errors.New("process command is required"))
	}
	switch arguments[0] {
	case "add":
		flags := newFlagSet("process add", stderr, mode)
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
		if mode.json {
			return writeJSON(stdout, processResult(process))
		}
		fmt.Fprintf(stdout, "process %s/%s added\n", process.ProjectID, process.ID)
		return nil
	case "remove":
		projectID, processID, err := parseProcessReference("process remove", arguments[1:], stderr, mode)
		if err != nil {
			return err
		}
		if err := client.RemoveProcess(ctx, projectID, processID); err != nil {
			return err
		}
		if mode.json {
			return writeJSON(stdout, removalResult{ProjectID: string(projectID), ProcessID: string(processID), Removed: true})
		}
		fmt.Fprintf(stdout, "process %s/%s removed\n", projectID, processID)
		return nil
	case "list":
		flags := newFlagSet("process list", stderr, mode)
		projectID := flags.String("project", "", "project ID")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		processes, err := client.ListProcesses(ctx, supervision.ProjectID(*projectID))
		if err != nil {
			return err
		}
		if mode.json {
			results := make([]processJSON, 0, len(processes))
			for _, process := range processes {
				results = append(results, processResult(process))
			}
			return writeJSON(stdout, results)
		}
		for _, process := range processes {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", process.ID, process.Command, strings.Join(process.Arguments, " "))
		}
		return nil
	case "start", "stop", "restart":
		projectID, processID, err := parseProcessReference("process "+arguments[0], arguments[1:], stderr, mode)
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
		if mode.json {
			return writeJSON(stdout, runtimeResult(runtimeState))
		}
		writeRuntime(stdout, runtimeState)
		return nil
	case "status":
		projectID, processID, err := parseProcessReference("process status", arguments[1:], stderr, mode)
		if err != nil {
			return err
		}
		runtimeState, exists, err := client.Status(ctx, projectID, processID)
		if err != nil {
			return err
		}
		if !exists {
			if mode.json {
				return writeJSON(stdout, runtimeJSON{ProjectID: string(projectID), ProcessID: string(processID), State: string(supervision.ProcessStateStopped)})
			}
			fmt.Fprintf(stdout, "%s/%s\t%s\n", projectID, processID, supervision.ProcessStateStopped)
			return nil
		}
		if mode.json {
			return writeJSON(stdout, runtimeResult(runtimeState))
		}
		writeRuntime(stdout, runtimeState)
		return nil
	case "logs":
		flags := newFlagSet("process logs", stderr, mode)
		projectID := flags.String("project", "", "project ID")
		processID := flags.String("process", "", "process ID")
		tailLimit := flags.Int("tail", 100, "maximum retained events")
		after := flags.Uint64("after", 0, "return events after this sequence")
		follow := flags.Bool("follow", false, "follow later output until interrupted")
		if err := parseExact(flags, arguments[1:]); err != nil {
			return err
		}
		if mode.json && *follow {
			return invalidArguments(errors.New("process logs --follow does not support --json"))
		}
		return runProcessLogs(ctx, client, supervision.ProjectID(*projectID), supervision.ProcessID(*processID), *after, *tailLimit, *follow, stdout, mode)
	default:
		return invalidArguments(fmt.Errorf("unknown process command %q", arguments[0]))
	}
}

func runProcessLogs(ctx context.Context, client operatorclient.Client, projectID supervision.ProjectID, processID supervision.ProcessID, after uint64, limit int, follow bool, stdout io.Writer, mode outputMode) error {
	first := true
	for {
		tail, err := client.Logs(ctx, projectID, processID, after, limit)
		if err != nil {
			if follow && ctx.Err() != nil {
				return nil
			}
			return err
		}
		if mode.json {
			return writeJSON(stdout, outputTailResult(tail))
		}
		if first && tail.Truncated {
			fmt.Fprintln(stdout, "[output truncated]")
		}
		for _, event := range tail.Events {
			writeOutputEvent(stdout, event)
			if event.Sequence > after {
				after = event.Sequence
			}
		}
		first = false
		if !follow {
			return nil
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}

func writeOutputEvent(writer io.Writer, event supervision.OutputEvent) {
	data := terminalstyle.RenderOutput(string(event.Data), interactiveOutput(writer))
	fmt.Fprintf(writer, "[%s] %s", event.Stream, data)
	if data == "" || data[len(data)-1] != '\n' {
		fmt.Fprintln(writer)
	}
}

func interactiveOutput(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok || !term.IsTerminal(file.Fd()) {
		return false
	}
	value, present := os.LookupEnv("NO_COLOR")
	return !present || value == ""
}

func parseProcessReference(name string, arguments []string, stderr io.Writer, mode outputMode) (supervision.ProjectID, supervision.ProcessID, error) {
	flags := newFlagSet(name, stderr, mode)
	projectID := flags.String("project", "", "project ID")
	processID := flags.String("process", "", "process ID")
	if err := parseExact(flags, arguments); err != nil {
		return "", "", err
	}
	return supervision.ProjectID(*projectID), supervision.ProcessID(*processID), nil
}

func parseExact(flags *flag.FlagSet, arguments []string) error {
	if err := flags.Parse(arguments); err != nil {
		return invalidArguments(err)
	}
	if flags.NArg() != 0 {
		return invalidArguments(fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " ")))
	}
	return nil
}

func newFlagSet(name string, stderr io.Writer, mode outputMode) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	if mode.json {
		flags.SetOutput(io.Discard)
	} else {
		flags.SetOutput(stderr)
	}
	return flags
}

func parseEnvironment(values []string) (map[string]string, error) {
	overrides := make(map[string]string, len(values))
	for _, value := range values {
		key, content, found := strings.Cut(value, "=")
		if !found || key == "" {
			return nil, invalidArguments(errors.New("invalid environment override, want KEY=VALUE"))
		}
		overrides[key] = content
	}
	return overrides, nil
}

func classifyClientError(err error) *commandError {
	var classified *commandError
	if errors.As(err, &classified) {
		return classified
	}
	var protocolError *control.ProtocolError
	if errors.As(err, &protocolError) {
		return &commandError{code: protocolError.Code, message: protocolError.Message, cause: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &commandError{code: "timeout", message: "the local daemon did not respond in time", cause: err}
	}
	if strings.Contains(err.Error(), "incompatible protocol version") {
		return &commandError{code: "incompatible_version", message: err.Error(), cause: err}
	}
	return &commandError{code: "connection_error", message: err.Error(), cause: err}
}

func invalidArguments(err error) *commandError {
	return &commandError{code: "invalid_arguments", message: err.Error(), cause: err}
}

func writeFailure(stderr io.Writer, jsonOutput bool, err *commandError, exitCode int) int {
	if jsonOutput {
		_ = writeJSON(stderr, errorEnvelope{Error: errorJSON{Code: err.code, Message: err.message}})
	} else {
		fmt.Fprintf(stderr, "dovik: %s\n", err.message)
	}
	return exitCode
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func hasJSONFlag(arguments []string) bool {
	for _, argument := range arguments {
		if argument == "--json" || strings.HasPrefix(argument, "--json=") {
			return true
		}
	}
	return false
}

func writeRuntime(writer io.Writer, runtimeState supervision.ProcessRuntime) {
	pid := "-"
	if runtimeState.PID != nil {
		pid = strconv.Itoa(*runtimeState.PID)
	}
	fmt.Fprintf(writer, "%s/%s\t%s\tpid=%s\tinstance=%s\n", runtimeState.ProjectID, runtimeState.ProcessID, runtimeState.State, pid, runtimeState.InstanceID)
}

func writeUsage(writer io.Writer) {
	writeHelp(writer, nil)
}

func helpTopic(arguments []string) ([]string, bool) {
	topic := make([]string, 0, 2)
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if argument == "--" {
			return nil, false
		}
		if argument == "--help" || argument == "-h" {
			return topic, true
		}
		if argument == "--json" || strings.HasPrefix(argument, "--json=") || strings.HasPrefix(argument, "--endpoint=") {
			continue
		}
		if argument == "--endpoint" {
			index++
			continue
		}
		if !strings.HasPrefix(argument, "-") && len(topic) < 2 {
			topic = append(topic, argument)
		}
	}
	return nil, false
}

func writeHelp(writer io.Writer, topic []string) {
	if len(topic) == 0 {
		fmt.Fprintln(writer, "Dovik supervises local development processes through one local daemon.")
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "Usage:")
		fmt.Fprintln(writer, "  dovik")
		fmt.Fprintln(writer, "  dovik [--endpoint PATH] [--json] project|process COMMAND [OPTIONS]")
		fmt.Fprintln(writer, "  dovik [--endpoint PATH] tui")
		fmt.Fprintln(writer, "  dovik [--endpoint PATH] [--json] whoami")
		fmt.Fprintln(writer, "  dovik [--endpoint PATH] mcp")
		fmt.Fprintln(writer, "  dovik [--endpoint PATH] gh -- GH_ARGUMENTS")
		fmt.Fprintln(writer, "  dovik [--endpoint PATH] [--json] identity|policy|session|doctor [OPTIONS]")
		fmt.Fprintln(writer, "  dovik [--endpoint PATH] project configure [--root PATH]")
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "Global options:")
		fmt.Fprintln(writer, "  --endpoint PATH  Override the local daemon endpoint")
		fmt.Fprintln(writer, "  --json           Emit one JSON document for non-interactive commands")
		fmt.Fprintln(writer, "  --help, -h       Show help")
		fmt.Fprintln(writer, "  --version        Show the Dovik version")
		return
	}

	command := strings.Join(topic, " ")
	switch command {
	case "project":
		fmt.Fprintln(writer, "Usage: dovik project add|remove|list|configure [OPTIONS]")
	case "project add":
		fmt.Fprintln(writer, "Usage: dovik project add --id ID --root PATH")
	case "project remove":
		fmt.Fprintln(writer, "Usage: dovik project remove --id ID")
	case "project list":
		fmt.Fprintln(writer, "Usage: dovik project list")
	case "project configure":
		fmt.Fprintln(writer, "Usage: dovik project configure [--root PATH]")
	case "process":
		fmt.Fprintln(writer, "Usage: dovik process add|remove|list|start|stop|restart|status|logs [OPTIONS]")
	case "process add":
		fmt.Fprintln(writer, "Usage: dovik process add --project ID --id ID --command PATH [--arg VALUE] [--env KEY=VALUE]")
	case "process remove", "process start", "process stop", "process restart", "process status":
		fmt.Fprintf(writer, "Usage: dovik %s --project ID --process ID\n", command)
	case "process list":
		fmt.Fprintln(writer, "Usage: dovik process list --project ID")
	case "process logs":
		fmt.Fprintln(writer, "Usage: dovik process logs --project ID --process ID [--tail COUNT] [--follow]")
	case "tui":
		fmt.Fprintln(writer, "Usage: dovik tui")
	case "whoami":
		fmt.Fprintln(writer, "Usage: dovik [--json] whoami")
	case "mcp":
		fmt.Fprintln(writer, "Usage: dovik mcp")
	case "gh":
		fmt.Fprintln(writer, "Usage: dovik gh -- GH_ARGUMENTS")
	case "identity":
		fmt.Fprintln(writer, "Usage: dovik identity COMMAND [OPTIONS]")
	case "policy":
		fmt.Fprintln(writer, "Usage: dovik policy COMMAND [OPTIONS]")
	case "session":
		fmt.Fprintln(writer, "Usage: dovik session COMMAND [OPTIONS]")
	case "doctor":
		fmt.Fprintln(writer, "Usage: dovik doctor [OPTIONS]")
	default:
		fmt.Fprintf(writer, "Usage: dovik %s [OPTIONS]\n", command)
	}
}

type projectJSON struct {
	ID            string `json:"id"`
	RootDirectory string `json:"rootDirectory"`
}

type processJSON struct {
	ProjectID        string   `json:"projectId"`
	ID               string   `json:"id"`
	Command          string   `json:"command"`
	Arguments        []string `json:"arguments,omitempty"`
	WorkingDirectory string   `json:"workingDirectory,omitempty"`
}

type runtimeJSON struct {
	ProjectID         string     `json:"projectId"`
	ProcessID         string     `json:"processId"`
	InstanceID        string     `json:"instanceId,omitempty"`
	PID               *int       `json:"pid,omitempty"`
	State             string     `json:"state"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	FinishedAt        *time.Time `json:"finishedAt,omitempty"`
	ExitCode          *int       `json:"exitCode,omitempty"`
	TerminationReason string     `json:"terminationReason,omitempty"`
}

type outputEventJSON struct {
	Sequence   uint64    `json:"sequence"`
	CapturedAt time.Time `json:"capturedAt"`
	Stream     string    `json:"stream"`
	Data       string    `json:"data"`
}

type outputTailJSON struct {
	Events    []outputEventJSON `json:"events"`
	Truncated bool              `json:"truncated"`
}

type removalResult struct {
	ProjectID string `json:"projectId"`
	ProcessID string `json:"processId,omitempty"`
	Removed   bool   `json:"removed"`
}

type errorJSON struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error errorJSON `json:"error"`
}

func projectResult(project supervision.ProjectDefinition) projectJSON {
	return projectJSON{ID: string(project.ID), RootDirectory: project.RootDirectory}
}

func processResult(process supervision.ProcessDefinition) processJSON {
	return processJSON{
		ProjectID: string(process.ProjectID), ID: string(process.ID), Command: process.Command,
		Arguments: append([]string(nil), process.Arguments...), WorkingDirectory: process.WorkingDirectory,
	}
}

func runtimeResult(runtimeState supervision.ProcessRuntime) runtimeJSON {
	return runtimeJSON{
		ProjectID: string(runtimeState.ProjectID), ProcessID: string(runtimeState.ProcessID),
		InstanceID: string(runtimeState.InstanceID), PID: runtimeState.PID, State: string(runtimeState.State),
		StartedAt: runtimeState.StartedAt, FinishedAt: runtimeState.FinishedAt,
		ExitCode: runtimeState.ExitCode, TerminationReason: runtimeState.TerminationReason,
	}
}

func outputTailResult(tail supervision.OutputTail) outputTailJSON {
	result := outputTailJSON{Events: make([]outputEventJSON, 0, len(tail.Events)), Truncated: tail.Truncated}
	for _, event := range tail.Events {
		result.Events = append(result.Events, outputEventJSON{
			Sequence: event.Sequence, CapturedAt: event.CapturedAt, Stream: string(event.Stream), Data: string(event.Data),
		})
	}
	return result
}

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}
