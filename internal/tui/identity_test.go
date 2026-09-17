package tui

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/charmbracelet/x/ansi"
)

func (client *fakeClient) Identity(context.Context, identity.Request) (identity.Snapshot, error) {
	return identity.Snapshot{}, nil
}
func (client *fakeClient) ExecuteGH(context.Context, identity.ExecutionRequest, io.Writer, io.Writer) (int, error) {
	return 1, errors.New("not used")
}
func (client *fakeClient) RunAgent(context.Context, identity.ContainerRequest, io.Writer, io.Writer) (int, error) {
	return 1, errors.New("not used")
}

func TestIdentityPanelAndConfigurationNavigation(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	model.width = 100
	model.height = 28
	model.registry = registryEmpty
	model.identities = identity.Snapshot{State: identity.State{Personas: []identity.Persona{{ID: "work", Name: "Work", Host: "github.com", Account: "example"}}, Projects: []identity.Project{{ID: "p", Name: "Example project", Persona: "work", Root: "/workspace", Repository: "owner/repo", Mode: "proxy-level", Policy: identity.Policy{Preset: "read-only"}}}}}
	opened, cmd := updateModel(t, model, tea.KeyPressMsg{Code: 'i'})
	if cmd == nil {
		t.Fatal("identity view did not refresh")
	}
	loading, _ := updateModel(t, opened, key("enter"))
	if loading.editor != nil {
		t.Fatal("configuration opened before project identity loaded")
	}
	opened, _ = updateModel(t, opened, identityLoadedMsg{snapshot: model.identities})
	view := ansi.Strip(opened.View().Content)
	for _, text := range []string{"processes (0/0)", "projects (1)", "personas (1)", "Example project", "github.com/owner/repo", "@example", "Read only", "repository operations", "● read", "○ comment", "persona switching", "not allowed", "agent sessions", "no isolated sessions", "enter edit", "l refresh"} {
		if !strings.Contains(view, text) {
			t.Fatalf("missing %q: %s", text, view)
		}
	}
	if strings.Contains(view, "read allowed") || strings.Contains(view, "comment blocked") {
		t.Fatalf("operation markers repeat their state in text: %s", view)
	}
	for _, unwanted := range []string{"Project 1/1:", "Protection: proxy-level", "Sessions: 0", "Allowed: read"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("identity view still contains raw diagnostic row %q: %s", unwanted, view)
		}
	}
	for _, operation := range identity.RepositoryOperations {
		if !strings.Contains(view, operation+" ") {
			t.Fatalf("project details omitted operation %q: %s", operation, view)
		}
	}
	configured, command := updateModel(t, opened, key("enter"))
	if configured.editor == nil || configured.editor.kind != editorProject || configured.editor.project.Root != "/workspace" || command == nil {
		t.Fatal("wrong configuration target")
	}
}

func TestIdentityAndPersonaNavigationUsesArrowsRefreshAndEnter(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	model.registry = registryEmpty
	model.activeWorkspace = workspaceProjects
	model.identityLoading = false
	model.identities = identity.Snapshot{State: identity.State{
		Personas: []identity.Persona{{ID: "one", Name: "One"}, {ID: "two", Name: "Two"}},
		Projects: []identity.Project{{ID: "one", Name: "One", Root: "/one"}, {ID: "two", Name: "Two", Root: "/two"}},
	}}

	unchanged, command := updateModel(t, model, key("j"))
	if unchanged.identitySelected != 0 || command != nil {
		t.Fatal("j still changes identity selection")
	}
	selected, _ := updateModel(t, unchanged, key("down"))
	if selected.identitySelected != 1 {
		t.Fatal("down did not change identity selection")
	}
	refreshed, command := updateModel(t, selected, key("l"))
	if !refreshed.identityLoading || command == nil {
		t.Fatal("l did not refresh identity")
	}
	selected.identityLoading = false
	edited, command := updateModel(t, selected, key("enter"))
	if edited.editor == nil || edited.editor.kind != editorProject || edited.editor.project.Root != "/two" || command == nil {
		t.Fatal("enter did not edit selected identity")
	}

	model.activeWorkspace = workspacePersonas
	model.identityLoading = false
	unchanged, command = updateModel(t, model, key("k"))
	if unchanged.personaSelected != 0 || command != nil {
		t.Fatal("k still changes persona selection")
	}
	selected, _ = updateModel(t, unchanged, key("down"))
	edited, command = updateModel(t, selected, key("enter"))
	if edited.editor == nil || edited.editor.kind != editorPersona || edited.editor.persona.ID != "two" || command == nil {
		t.Fatal("enter did not edit selected persona")
	}
}

func TestWorkspaceTabsWrapAndPreserveSelections(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	model.width = 100
	model.height = 28
	model.registry = registryEmpty
	model.identitySelected = 1
	model.personaSelected = 1
	snapshot := identity.Snapshot{State: identity.State{
		Personas: []identity.Persona{{ID: "one", Name: "One"}, {ID: "two", Name: "Two"}},
		Projects: []identity.Project{{ID: "one", Name: "One"}, {ID: "two", Name: "Two"}},
	}}

	personasTab, command := updateModel(t, model, key("right"))
	if personasTab.activeWorkspace != workspacePersonas || command == nil {
		t.Fatal("right did not wrap to and refresh personas")
	}
	personasTab, _ = updateModel(t, personasTab, identityLoadedMsg{snapshot: snapshot})
	identityTab, command := updateModel(t, personasTab, key("right"))
	if identityTab.activeWorkspace != workspaceProjects || command == nil {
		t.Fatal("right did not open and refresh projects")
	}
	identityTab, _ = updateModel(t, identityTab, identityLoadedMsg{snapshot: snapshot})
	processesTab, command := updateModel(t, identityTab, key("right"))
	if processesTab.activeWorkspace != workspaceProcesses || command != nil {
		t.Fatal("right did not wrap to processes")
	}
	wrapped, command := updateModel(t, processesTab, key("left"))
	if wrapped.activeWorkspace != workspaceProjects || command == nil || wrapped.identitySelected != 1 || wrapped.personaSelected != 1 {
		t.Fatal("left wrap lost workspace state")
	}
}

func TestPersonaPanelUsesPublicMetadataAndAssignments(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	model.width = 100
	model.height = 28
	model.registry = registryEmpty
	model.activeWorkspace = workspacePersonas
	model.identityLoading = false
	model.identities = identity.Snapshot{State: identity.State{
		Personas: []identity.Persona{{ID: "personal", Name: "Personal", Host: "github.com", Account: "maxie", GitName: "Maxie", GitEmail: "maxie@example.test"}},
		Projects: []identity.Project{{ID: "dovik", Name: "Dovik", Persona: "personal"}},
	}}

	view := ansi.Strip(model.View().Content)
	for _, text := range []string{"processes (0/0)", "projects (1)", "personas (1)", "persona", "Personal", "persona id", "personal", "@maxie", "github.com", "git author name", "Maxie", "git author email", "maxie@example.test", "assigned projects", "Dovik"} {
		if !strings.Contains(view, text) {
			t.Fatalf("missing %q: %s", text, view)
		}
	}
}

func TestCompactIdentityViewsKeepCompleteStatusLists(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	model.width, model.height = 80, 24
	model.registry = registryEmpty
	model.identityLoading = false
	model.identities = identity.Snapshot{State: identity.State{
		Personas: []identity.Persona{{ID: "personal", Name: "Personal", Host: "github.com", Account: "maxie", GitName: "Maxie", GitEmail: "maxie@example.test"}},
		Projects: []identity.Project{
			{ID: "dovik", Name: "Dovik", Persona: "personal", Repository: "owner/dovik", Mode: "proxy-level", Policy: identity.Policy{Preset: "maintain"}},
			{ID: "cadder", Name: "Cadder", Persona: "personal"},
			{ID: "arcantry", Name: "Arcantry", Persona: "personal"},
			{ID: "ttyglass", Name: "ttyglass", Persona: "personal"},
		},
	}}

	model.activeWorkspace = workspaceProjects
	projectView := ansi.Strip(model.View().Content)
	for _, operation := range identity.RepositoryOperations {
		if !strings.Contains(projectView, operation+" ") {
			t.Fatalf("compact project view omitted operation %q:\n%s", operation, projectView)
		}
	}

	model.activeWorkspace = workspacePersonas
	personaView := ansi.Strip(model.View().Content)
	for _, project := range []string{"Dovik", "Cadder", "Arcantry", "ttyglass"} {
		if !strings.Contains(personaView, project) {
			t.Fatalf("compact persona view omitted assignment %q:\n%s", project, personaView)
		}
	}
}

func TestEmbeddedProjectEditorKeepsFieldsTogetherAndUsesDistinctConfirmation(t *testing.T) {
	model := NewModel(context.Background(), &fakeClient{})
	model.width, model.height = 100, 40
	model.registry = registryEmpty
	model.activeWorkspace = workspaceProjects
	model.identityLoading = false
	model.identities = identity.Snapshot{State: identity.State{
		Version:  1,
		Personas: []identity.Persona{{ID: "personal", Name: "Personal", Host: "github.com", Account: "maxie", GitName: "Maxie", GitEmail: "maxie@example.test"}},
		Projects: []identity.Project{{ID: "dovik", Name: "Dovik", Description: "Local process manager", Root: `D:\Projects\Personal\dovik`, Repository: "MrMaxie/dovik", Persona: "personal", Mode: "proxy-level", Policy: identity.Policy{Preset: "maintain", Exceptions: map[string]bool{}}}},
	}}

	edited, command := updateModel(t, model, key("enter"))
	if command == nil || edited.editor == nil || edited.editor.kind != editorProject {
		t.Fatal("project editor did not open inside the model")
	}
	_, _ = edited.editor.form.Update(tea.WindowSizeMsg{Width: edited.editorWidth(), Height: edited.editorHeight()})
	view := ansi.Strip(edited.View().Content)
	if lipgloss.Width(edited.View().Content) != model.width || lipgloss.Height(edited.View().Content) != model.height {
		t.Fatalf("embedded editor dimensions = %dx%d, want %dx%d", lipgloss.Width(edited.View().Content), lipgloss.Height(edited.View().Content), model.width, model.height)
	}
	for _, label := range []string{"edit project", "project id", "project name", "description", "github repository", "persona", "protection"} {
		if !strings.Contains(view, label) {
			t.Fatalf("single project form is missing %q:\n%s", label, view)
		}
	}
	if !strings.Contains(view, "│  │ Dovik") {
		t.Fatalf("focused text value is not separated from its focus marker:\n%s", view)
	}
	if strings.Contains(view, "dovik edit project") || !strings.Contains(view, "┌") || !strings.Contains(view, "└") {
		t.Fatalf("project editor is not one clearly focused in-place form:\n%s", view)
	}
	lines := strings.Split(view, "\n")
	descriptionLine, repositoryLine := -1, -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "description") && descriptionLine < 0 {
			descriptionLine = index
		}
		if strings.Contains(trimmed, "github repository") && repositoryLine < 0 {
			repositoryLine = index
		}
	}
	if descriptionLine < 0 || repositoryLine < 0 || repositoryLine-descriptionLine > 6 {
		t.Fatalf("description field reserves excessive vertical space:\n%s", view)
	}

	edited.confirmation = &confirmationState{kind: editorProject}
	confirmation := ansi.Strip(edited.View().Content)
	if !strings.Contains(confirmation, "confirm changes") || !strings.Contains(confirmation, "cancel") || !strings.Contains(confirmation, "save") {
		t.Fatalf("project confirmation is not a distinct modal:\n%s", confirmation)
	}
	cancelled, _ := updateModel(t, edited, key("esc"))
	if cancelled.editor != nil || cancelled.confirmation != nil {
		t.Fatal("escape did not cancel the embedded editor")
	}
}
