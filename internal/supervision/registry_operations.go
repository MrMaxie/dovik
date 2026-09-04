package supervision

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// AddProject validates and persists one project definition.
func (manager *LifecycleManager) AddProject(definition ProjectDefinition) (ProjectDefinition, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if definition.ID == "" {
		return ProjectDefinition{}, fmt.Errorf("project ID is required")
	}
	if !filepath.IsAbs(definition.RootDirectory) {
		return ProjectDefinition{}, fmt.Errorf("project root must be absolute")
	}
	info, err := os.Stat(definition.RootDirectory)
	if err != nil {
		return ProjectDefinition{}, fmt.Errorf("inspect project root: %w", err)
	}
	if !info.IsDir() {
		return ProjectDefinition{}, fmt.Errorf("project root is not a directory")
	}
	if manager.registry.Projects[definition.ID] != nil {
		return ProjectDefinition{}, fmt.Errorf("project %q already exists", definition.ID)
	}

	manager.registry.Projects[definition.ID] = &RegisteredProject{
		Definition: definition,
		Processes:  make(map[ProcessID]*RegisteredProcess),
	}
	if err := manager.persister.Save(manager.registry); err != nil {
		delete(manager.registry.Projects, definition.ID)
		return ProjectDefinition{}, fmt.Errorf("persist project: %w", err)
	}
	return definition, nil
}

// RemoveProject removes an empty project definition.
func (manager *LifecycleManager) RemoveProject(projectID ProjectID) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	project := manager.registry.Projects[projectID]
	if project == nil {
		return fmt.Errorf("project %q not found", projectID)
	}
	if len(project.Processes) != 0 {
		return fmt.Errorf("project %q still owns process definitions", projectID)
	}
	delete(manager.registry.Projects, projectID)
	if err := manager.persister.Save(manager.registry); err != nil {
		manager.registry.Projects[projectID] = project
		return fmt.Errorf("persist project removal: %w", err)
	}
	return nil
}

// ListProjects returns project definitions ordered by ID.
func (manager *LifecycleManager) ListProjects() []ProjectDefinition {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	projects := make([]ProjectDefinition, 0, len(manager.registry.Projects))
	for _, project := range manager.registry.Projects {
		projects = append(projects, project.Definition)
	}
	sort.Slice(projects, func(left int, right int) bool { return projects[left].ID < projects[right].ID })
	return projects
}

// AddProcess validates and persists one process definition. The returned value
// omits environment overrides so callers cannot expose their values.
func (manager *LifecycleManager) AddProcess(definition ProcessDefinition) (ProcessDefinition, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	project := manager.registry.Projects[definition.ProjectID]
	if project == nil {
		return ProcessDefinition{}, fmt.Errorf("project %q not found", definition.ProjectID)
	}
	if definition.ID == "" {
		return ProcessDefinition{}, fmt.Errorf("process ID is required")
	}
	if definition.Command == "" {
		return ProcessDefinition{}, fmt.Errorf("process command is required")
	}
	if project.Processes[definition.ID] != nil {
		return ProcessDefinition{}, fmt.Errorf("process %q already exists in project %q", definition.ID, definition.ProjectID)
	}
	if definition.WorkingDirectory != "" {
		workingDirectory := definition.WorkingDirectory
		if !filepath.IsAbs(workingDirectory) {
			workingDirectory = filepath.Join(project.Definition.RootDirectory, workingDirectory)
		}
		info, err := os.Stat(workingDirectory)
		if err != nil {
			return ProcessDefinition{}, fmt.Errorf("inspect process working directory: %w", err)
		}
		if !info.IsDir() {
			return ProcessDefinition{}, fmt.Errorf("process working directory is not a directory")
		}
	}

	definition.Arguments = cloneStrings(definition.Arguments)
	definition.EnvironmentOverrides = cloneStringMap(definition.EnvironmentOverrides)
	project.Processes[definition.ID] = &RegisteredProcess{Definition: definition}
	if err := manager.persister.Save(manager.registry); err != nil {
		delete(project.Processes, definition.ID)
		return ProcessDefinition{}, fmt.Errorf("persist process: %w", err)
	}
	return publicProcessDefinition(definition), nil
}

// RemoveProcess removes an inactive process definition.
func (manager *LifecycleManager) RemoveProcess(projectID ProjectID, processID ProcessID) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	_, process, err := manager.findProcess(processKey{projectID: projectID, processID: processID})
	if err != nil {
		return err
	}
	if process.Runtime != nil && process.Runtime.State.IsActive() {
		return fmt.Errorf("process %q in project %q is active", processID, projectID)
	}
	project := manager.registry.Projects[projectID]
	delete(project.Processes, processID)
	if err := manager.persister.Save(manager.registry); err != nil {
		project.Processes[processID] = process
		return fmt.Errorf("persist process removal: %w", err)
	}
	delete(manager.outputs, processKey{projectID: projectID, processID: processID})
	return nil
}

// ListProcesses returns process definitions ordered by ID without environment
// override values.
func (manager *LifecycleManager) ListProcesses(projectID ProjectID) ([]ProcessDefinition, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	project := manager.registry.Projects[projectID]
	if project == nil {
		return nil, fmt.Errorf("project %q not found", projectID)
	}
	processes := make([]ProcessDefinition, 0, len(project.Processes))
	for _, process := range project.Processes {
		processes = append(processes, publicProcessDefinition(process.Definition))
	}
	sort.Slice(processes, func(left int, right int) bool { return processes[left].ID < processes[right].ID })
	return processes, nil
}

func publicProcessDefinition(definition ProcessDefinition) ProcessDefinition {
	definition.Arguments = cloneStrings(definition.Arguments)
	definition.EnvironmentOverrides = nil
	return definition
}
