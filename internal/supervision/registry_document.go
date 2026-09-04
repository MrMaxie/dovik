package supervision

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

const registryDocumentVersion = 1

type registryDocument struct {
	Version  int               `json:"version"`
	Projects []projectDocument `json:"projects"`
}

type projectDocument struct {
	ID            string            `json:"id"`
	RootDirectory string            `json:"rootDirectory"`
	Processes     []processDocument `json:"processes"`
}

type processDocument struct {
	ID                   string            `json:"id"`
	Command              string            `json:"command"`
	Arguments            []string          `json:"arguments,omitempty"`
	WorkingDirectory     string            `json:"workingDirectory,omitempty"`
	EnvironmentOverrides map[string]string `json:"environmentOverrides,omitempty"`
	Runtime              *runtimeDocument  `json:"runtime,omitempty"`
}

type runtimeDocument struct {
	InstanceID        string       `json:"instanceId"`
	PID               *int         `json:"pid,omitempty"`
	State             ProcessState `json:"state"`
	StartedAt         *time.Time   `json:"startedAt,omitempty"`
	FinishedAt        *time.Time   `json:"finishedAt,omitempty"`
	ExitCode          *int         `json:"exitCode,omitempty"`
	TerminationReason string       `json:"terminationReason,omitempty"`
}

func registryToDocument(registry *Registry) (registryDocument, error) {
	document := registryDocument{Version: registryDocumentVersion}
	if registry == nil {
		return document, fmt.Errorf("registry is nil")
	}

	projectIDs := make([]string, 0, len(registry.Projects))
	for projectID := range registry.Projects {
		projectIDs = append(projectIDs, string(projectID))
	}
	sort.Strings(projectIDs)

	for _, projectIDText := range projectIDs {
		projectID := ProjectID(projectIDText)
		project := registry.Projects[projectID]
		if err := validateRegisteredProject(projectID, project); err != nil {
			return registryDocument{}, err
		}

		projectOutput := projectDocument{
			ID:            projectIDText,
			RootDirectory: project.Definition.RootDirectory,
		}
		processIDs := make([]string, 0, len(project.Processes))
		for processID := range project.Processes {
			processIDs = append(processIDs, string(processID))
		}
		sort.Strings(processIDs)

		for _, processIDText := range processIDs {
			processID := ProcessID(processIDText)
			process := project.Processes[processID]
			if err := validateRegisteredProcess(projectID, processID, process); err != nil {
				return registryDocument{}, err
			}
			processOutput := processDocument{
				ID:                   processIDText,
				Command:              process.Definition.Command,
				Arguments:            cloneStrings(process.Definition.Arguments),
				WorkingDirectory:     process.Definition.WorkingDirectory,
				EnvironmentOverrides: cloneStringMap(process.Definition.EnvironmentOverrides),
				Runtime:              runtimeToDocument(process.Runtime),
			}
			projectOutput.Processes = append(projectOutput.Processes, processOutput)
		}
		document.Projects = append(document.Projects, projectOutput)
	}

	return document, nil
}

func documentToRegistry(document registryDocument) (*Registry, error) {
	if document.Version != registryDocumentVersion {
		return nil, fmt.Errorf("unsupported registry version %d", document.Version)
	}

	registry := NewRegistry()
	for _, projectInput := range document.Projects {
		projectID := ProjectID(projectInput.ID)
		if projectID == "" {
			return nil, fmt.Errorf("project ID is empty")
		}
		if !filepath.IsAbs(projectInput.RootDirectory) {
			return nil, fmt.Errorf("project %q root is not absolute", projectID)
		}
		if _, exists := registry.Projects[projectID]; exists {
			return nil, fmt.Errorf("duplicate project %q", projectID)
		}

		project := &RegisteredProject{
			Definition: ProjectDefinition{ID: projectID, RootDirectory: projectInput.RootDirectory},
			Processes:  make(map[ProcessID]*RegisteredProcess),
		}
		for _, processInput := range projectInput.Processes {
			processID := ProcessID(processInput.ID)
			if processID == "" {
				return nil, fmt.Errorf("process ID is empty in project %q", projectID)
			}
			if processInput.Command == "" {
				return nil, fmt.Errorf("process %q in project %q has an empty command", processID, projectID)
			}
			if _, exists := project.Processes[processID]; exists {
				return nil, fmt.Errorf("duplicate process %q in project %q", processID, projectID)
			}

			process := &RegisteredProcess{
				Definition: ProcessDefinition{
					ProjectID:            projectID,
					ID:                   processID,
					Command:              processInput.Command,
					Arguments:            cloneStrings(processInput.Arguments),
					WorkingDirectory:     processInput.WorkingDirectory,
					EnvironmentOverrides: cloneStringMap(processInput.EnvironmentOverrides),
				},
				Runtime: documentToRuntime(projectID, processID, processInput.Runtime),
			}
			if process.Runtime != nil && !process.Runtime.State.IsValid() {
				return nil, fmt.Errorf("process %q in project %q has invalid runtime state %q", processID, projectID, process.Runtime.State)
			}
			project.Processes[processID] = process
		}
		registry.Projects[projectID] = project
	}

	return registry, nil
}

func validateRegisteredProject(projectID ProjectID, project *RegisteredProject) error {
	if projectID == "" || project == nil || project.Definition.ID != projectID {
		return fmt.Errorf("invalid project record %q", projectID)
	}
	if !filepath.IsAbs(project.Definition.RootDirectory) {
		return fmt.Errorf("project %q root is not absolute", projectID)
	}
	return nil
}

func validateRegisteredProcess(projectID ProjectID, processID ProcessID, process *RegisteredProcess) error {
	if processID == "" || process == nil || process.Definition.ID != processID || process.Definition.ProjectID != projectID {
		return fmt.Errorf("invalid process record %q in project %q", processID, projectID)
	}
	if process.Definition.Command == "" {
		return fmt.Errorf("process %q in project %q has an empty command", processID, projectID)
	}
	if process.Runtime != nil {
		if process.Runtime.ProjectID != projectID || process.Runtime.ProcessID != processID || process.Runtime.InstanceID == "" {
			return fmt.Errorf("invalid runtime for process %q in project %q", processID, projectID)
		}
		if !process.Runtime.State.IsValid() {
			return fmt.Errorf("process %q in project %q has invalid runtime state %q", processID, projectID, process.Runtime.State)
		}
		if process.Runtime.PID != nil && *process.Runtime.PID <= 0 {
			return fmt.Errorf("process %q in project %q has invalid PID", processID, projectID)
		}
	}
	return nil
}

func runtimeToDocument(runtime *ProcessRuntime) *runtimeDocument {
	if runtime == nil {
		return nil
	}
	return &runtimeDocument{
		InstanceID:        string(runtime.InstanceID),
		PID:               cloneInt(runtime.PID),
		State:             runtime.State,
		StartedAt:         cloneTime(runtime.StartedAt),
		FinishedAt:        cloneTime(runtime.FinishedAt),
		ExitCode:          cloneInt(runtime.ExitCode),
		TerminationReason: runtime.TerminationReason,
	}
}

func documentToRuntime(projectID ProjectID, processID ProcessID, document *runtimeDocument) *ProcessRuntime {
	if document == nil {
		return nil
	}
	return &ProcessRuntime{
		ProjectID:         projectID,
		ProcessID:         processID,
		InstanceID:        RuntimeInstanceID(document.InstanceID),
		PID:               cloneInt(document.PID),
		State:             document.State,
		StartedAt:         cloneTime(document.StartedAt),
		FinishedAt:        cloneTime(document.FinishedAt),
		ExitCode:          cloneInt(document.ExitCode),
		TerminationReason: document.TerminationReason,
	}
}

func cloneStrings(input []string) []string {
	return append([]string(nil), input...)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneInt(input *int) *int {
	if input == nil {
		return nil
	}
	output := *input
	return &output
}

func cloneTime(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	output := *input
	return &output
}
