package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/identityui"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/charmbracelet/x/term"
)

func runIdentityCommand(ctx context.Context, client operatorclient.Client, args []string, stdin io.Reader, stdout, stderr io.Writer, mode outputMode) error {
	if len(args) < 2 {
		return invalidArguments(fmt.Errorf("identity command required: list, put, import, status, use, gh"))
	}
	flags := newFlagSet("identity "+args[1], stderr, mode)
	id := flags.String("id", "", "persona ID")
	name := flags.String("name", "", "display name")
	gitName := flags.String("git-name", "", "Git author")
	email := flags.String("email", "", "Git email")
	host := flags.String("host", "github.com", "GitHub host")
	account := flags.String("account", "", "GitHub account")
	path := flags.String("path", "", "original gh executable")
	file := flags.String("file", "", "legacy persona file")
	root := flags.String("root", ".", "project directory")
	projectID := flags.String("project", "", "project ID")
	confirm := flags.Bool("confirm", false, "confirm persona import")
	short := flags.Bool("short", false, "print persona label only")
	applyGit := flags.Bool("apply-git", false, "set project-local Git author")
	if err := parseExact(flags, args[2:]); err != nil {
		return err
	}
	request := identity.Request{Action: "get"}
	switch args[1] {
	case "list", "status", "use":
	case "put":
		request = identity.Request{Action: "persona.put", Persona: &identity.Persona{ID: *id, Name: *name, GitName: *gitName, GitEmail: *email, Host: *host, Account: *account}}
	case "gh":
		request = identity.Request{Action: "gh.set", Path: *path}
	case "import":
		data, err := readInputFile(*file)
		if err != nil {
			return fmt.Errorf("cannot read selected import file")
		}
		if len(data) > 1024*1024 {
			return fmt.Errorf("import file is too large")
		}
		personas, err := identity.PreviewImport(data, *host)
		if err != nil {
			return err
		}
		if !*confirm {
			return writeJSON(stdout, struct {
				Preview     []identity.Persona `json:"preview"`
				Instruction string             `json:"instruction"`
			}{personas, "Review account names, then repeat with --confirm. Use identity put to correct imported metadata."})
		}
		request = identity.Request{Action: "persona.import", Personas: personas}
	default:
		return invalidArguments(fmt.Errorf("unknown identity command"))
	}
	snapshot, err := client.Identity(ctx, request)
	if err != nil {
		return err
	}
	if args[1] == "status" || args[1] == "use" {
		resolved, err := identity.GitRoot(ctx, *root)
		if err != nil {
			return err
		}
		project, err := snapshot.State.FindProject(*projectID, resolved)
		if err != nil {
			if *short {
				_, err = fmt.Fprintln(stdout, "?")
				return err
			}
			return err
		}
		personaID := project.Persona
		if args[1] == "use" {
			personaID = *id
		}
		persona, err := snapshot.State.FindPersona(personaID)
		if err != nil {
			return err
		}
		if args[1] == "use" {
			project.Persona = persona.ID
			if _, err := client.Identity(ctx, identity.Request{Action: "project.put", Project: &project}); err != nil {
				return err
			}
		}
		if *applyGit {
			if err := identity.ApplyGitAuthor(ctx, resolved, persona); err != nil {
				return err
			}
		}
		if *short {
			_, err = fmt.Fprintln(stdout, persona.Name)
			return err
		}
		return writeJSON(stdout, identity.AgentContext{Project: project, Persona: persona})
	}
	if mode.json {
		return writeJSON(stdout, snapshot)
	}
	for _, p := range snapshot.State.Personas {
		fmt.Fprintf(stdout, "%s\t%s\t%s/%s\n", p.ID, p.Name, p.Host, p.Account)
	}
	return nil
}

func runPolicyCommand(ctx context.Context, client operatorclient.Client, args []string, stdout, stderr io.Writer, mode outputMode) error {
	if len(args) < 2 {
		return invalidArguments(fmt.Errorf("policy command required: show or set"))
	}
	flags := newFlagSet("policy", stderr, mode)
	projectID := flags.String("project", "", "project ID")
	preset := flags.String("preset", "", "read-only, collaborate, maintain")
	var allow, deny stringList
	flags.Var(&allow, "allow", "permit an operation")
	flags.Var(&deny, "deny", "deny an operation")
	if err := parseExact(flags, args[2:]); err != nil {
		return err
	}
	request := identity.Request{Action: "get"}
	if args[1] == "set" {
		policy := identity.Policy{Preset: *preset, Exceptions: map[string]bool{}}
		for _, op := range allow {
			policy.Exceptions[op] = true
		}
		for _, op := range deny {
			policy.Exceptions[op] = false
		}
		request = identity.Request{Action: "policy.set", ProjectID: *projectID, Policy: &policy}
	} else if args[1] == "enable" || args[1] == "disable" {
		snapshot, err := client.Identity(ctx, request)
		if err != nil {
			return err
		}
		project, err := snapshot.State.FindProject(*projectID, "")
		if err != nil {
			return err
		}
		if project.Mode == "agent-isolation" && args[1] == "disable" {
			return fmt.Errorf("isolated projects cannot bypass the execution policy")
		}
		project.ProxyEnabled = args[1] == "enable"
		request = identity.Request{Action: "project.put", Project: &project}
	} else if args[1] != "show" {
		return invalidArguments(fmt.Errorf("unknown policy command"))
	}
	snapshot, err := client.Identity(ctx, request)
	if err != nil {
		return err
	}
	project, err := snapshot.State.FindProject(*projectID, "")
	if err != nil {
		return err
	}
	return writeJSON(stdout, struct {
		Policy       identity.Policy `json:"policy"`
		ProxyEnabled bool            `json:"proxyEnabled"`
	}{project.Policy, project.ProxyEnabled})
}

func runSessionCommand(ctx context.Context, client operatorclient.Client, args []string, stdout, stderr io.Writer, mode outputMode) error {
	if len(args) < 2 {
		return invalidArguments(fmt.Errorf("session command required: create, list, revoke, persona, context"))
	}
	flags := newFlagSet("session", stderr, mode)
	id := flags.String("id", "", "session ID")
	project := flags.String("project", "", "project ID")
	backend := flags.String("backend", "proxy-level", "proxy-level, native, docker, podman")
	principal := flags.String("principal", "", "native agent account")
	persona := flags.String("persona", "", "persona ID")
	if err := parseExact(flags, args[2:]); err != nil {
		return err
	}
	if endpoint := os.Getenv("DOVIK_AGENT_ENDPOINT"); endpoint != "" {
		action := args[1]
		if action != "context" && action != "persona" {
			return fmt.Errorf("session administration requires the operator terminal")
		}
		agentContext, err := callAgentContext(ctx, endpoint, action, *persona)
		if err != nil {
			return err
		}
		return writeJSON(stdout, agentContext)
	}
	action := "get"
	switch args[1] {
	case "list", "context":
	case "create", "revoke", "persona":
		action = "session." + args[1]
	default:
		return invalidArguments(fmt.Errorf("unknown session command"))
	}
	snapshot, err := client.Identity(ctx, identity.Request{Action: action, ProjectID: *project, SessionID: *id, Backend: *backend, Principal: *principal, PersonaID: *persona})
	if err != nil {
		return err
	}
	return writeJSON(stdout, snapshot.Sessions)
}

func runDoctor(ctx context.Context, client operatorclient.Client, args []string, stdout, stderr io.Writer, mode outputMode) error {
	flags := newFlagSet("doctor", stderr, mode)
	principal := flags.String("principal", "", "native account to inspect")
	backend := flags.String("backend", "", "container runtime to inspect")
	session := flags.String("session", "", "session channel to inspect")
	if err := parseExact(flags, args); err != nil {
		return err
	}
	snapshot, err := client.Identity(ctx, identity.Request{Action: "doctor", Principal: *principal, Backend: *backend, SessionID: *session})
	if err != nil {
		return err
	}
	if err := writeJSON(stdout, snapshot.Checks); err != nil {
		return err
	}
	for _, check := range snapshot.Checks {
		if !check.Ready {
			return fmt.Errorf("doctor found unmet requirements")
		}
	}
	return nil
}

func readInputFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil || len(data) > 1024*1024 {
		return nil, fmt.Errorf("input exceeds 1 MiB or cannot be read")
	}
	return data, nil
}

func interactive(input io.Reader, output io.Writer) bool {
	inputFile, inputOK := input.(*os.File)
	outputFile, outputOK := output.(*os.File)
	return inputOK && outputOK && term.IsTerminal(inputFile.Fd()) && term.IsTerminal(outputFile.Fd())
}

func runProjectConfigure(ctx context.Context, client operatorclient.Client, args []string, input io.Reader, output, stderr io.Writer, mode outputMode) error {
	flags := newFlagSet("project configure", stderr, mode)
	root := flags.String("root", ".", "project directory")
	file := flags.String("file", "", "operator configuration JSON file")
	if err := parseExact(flags, args); err != nil {
		return err
	}
	if *file == "" {
		if mode.json || !interactive(input, output) {
			return fmt.Errorf("use an operator terminal or supply --file with explicit configuration")
		}
		return identityui.Configure(ctx, client, *root, input, output)
	}
	data, err := readInputFile(*file)
	if err != nil {
		return err
	}
	var config struct {
		Persona identity.Persona `json:"persona"`
		Project identity.Project `json:"project"`
		GHPath  string           `json:"ghPath"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("invalid project configuration: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("configuration must contain one JSON object")
	}
	snapshot, err := client.Identity(ctx, identity.Request{Action: "configure", Persona: &config.Persona, Project: &config.Project, Path: config.GHPath})
	if err != nil {
		return err
	}
	return writeJSON(output, snapshot)
}

func prepareInvocation(args []string, stdin io.Reader) (identity.Invocation, error) {
	in := identity.Invocation{Arguments: append([]string{}, args...)}
	if len(args) == 3 && args[0] == "auth" && args[1] == "git-credential" && args[2] == "get" {
		data, err := io.ReadAll(io.LimitReader(stdin, 64*1024+1))
		if err != nil || len(data) > 64*1024 {
			return in, fmt.Errorf("Git credential input exceeds 64 KiB or cannot be read")
		}
		in.Input = data
		return in, nil
	}
	for i := 0; i < len(in.Arguments); i++ {
		flag := in.Arguments[i]
		var path string
		if flag == "--body-file" || flag == "-F" {
			i++
			if i == len(in.Arguments) {
				return in, fmt.Errorf("body file argument is required")
			}
			path = in.Arguments[i]
			in.Arguments[i] = "-"
		} else if strings.HasPrefix(flag, "--body-file=") {
			path = strings.TrimPrefix(flag, "--body-file=")
			in.Arguments[i] = "--body-file=-"
		} else {
			continue
		}
		if in.Input != nil {
			return in, fmt.Errorf("only one body input is supported")
		}
		reader := stdin
		var file *os.File
		if path != "-" {
			var err error
			file, err = os.Open(filepath.Clean(path))
			if err != nil {
				return in, fmt.Errorf("cannot open body file")
			}
			defer file.Close()
			reader = file
		}
		data, err := io.ReadAll(io.LimitReader(reader, 1024*1024+1))
		if err != nil || len(data) > 1024*1024 {
			return in, fmt.Errorf("body input exceeds 1 MiB or cannot be read")
		}
		in.Input = data
	}
	return in, nil
}
