package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/terminalstyle"
)

type editorKind uint8

const (
	editorProcess editorKind = iota + 1
	editorProject
	editorPersona
)

type editorState struct {
	kind          editorKind
	form          *huh.Form
	processAction string
	project       identity.Project
	persona       identity.Persona
}

type confirmationState struct {
	kind    editorKind
	approve bool
}

type identitySavedMsg struct {
	kind     editorKind
	snapshot identity.Snapshot
	err      error
}

func (model Model) openProcessEditor() (tea.Model, tea.Cmd) {
	item := model.items[model.selected]
	action := ""
	options := []huh.Option[string]{}
	if model.statusKnown && model.currentState().IsActive() {
		action = "stop"
		options = append(options, huh.NewOption("stop", "stop"), huh.NewOption("restart", "restart"))
	} else {
		action = "start"
		options = append(options, huh.NewOption("start", "start"))
	}
	editor := &editorState{kind: editorProcess, processAction: action}
	editor.form = terminalstyle.NewEmbeddedForm(
		"",
		model.editorWidth(),
		model.editorHeight(),
		huh.NewSelect[string]().
			Title("action").
			Description("choose what to do with "+string(item.key.processID)+" in "+string(item.key.projectID)).
			Options(options...).
			Value(&editor.processAction),
	)
	model.editor = editor
	return model, editor.form.Init()
}

func (model Model) openProjectEditor() (tea.Model, tea.Cmd) {
	if len(model.identities.State.Projects) == 0 {
		return model, nil
	}
	project := model.identities.State.Projects[min(model.identitySelected, len(model.identities.State.Projects)-1)]
	personaOptions := make([]huh.Option[string], 0, len(model.identities.State.Personas))
	for _, persona := range model.identities.State.Personas {
		personaOptions = append(personaOptions, huh.NewOption(persona.Name+" (@"+persona.Account+")", persona.ID))
	}
	editor := &editorState{kind: editorProject, project: project}
	editor.form = terminalstyle.NewEmbeddedForm(
		"",
		model.editorWidth(),
		model.editorHeight(),
		huh.NewNote().Title("project id").Description(project.ID),
		huh.NewInput().Title("project name").Prompt("").Value(&editor.project.Name).Validate(requiredText("project name")),
		huh.NewText().Title("description").CharLimit(4096).Value(&editor.project.Description).WithHeight(3),
		huh.NewInput().Title("github repository").Description("OWNER/REPO").Prompt("").Value(&editor.project.Repository).Validate(validateRepository),
		huh.NewSelect[string]().Title("persona").Options(personaOptions...).Value(&editor.project.Persona),
		huh.NewSelect[string]().Title("protection").Options(
			huh.NewOption("proxy-level - transparently route the selected persona", "proxy-level"),
			huh.NewOption("agent isolation - enforce the saved policy in an isolated session", "agent-isolation"),
		).Value(&editor.project.Mode),
	)
	model.editor = editor
	return model, editor.form.Init()
}

func (model Model) openPersonaEditor() (tea.Model, tea.Cmd) {
	if len(model.identities.State.Personas) == 0 {
		return model, nil
	}
	persona := model.identities.State.Personas[min(model.personaSelected, len(model.identities.State.Personas)-1)]
	editor := &editorState{kind: editorPersona, persona: persona}
	editor.form = terminalstyle.NewEmbeddedForm(
		"",
		model.editorWidth(),
		model.editorHeight(),
		huh.NewNote().Title("persona id").Description(persona.ID),
		huh.NewInput().Title("persona name").Prompt("").Value(&editor.persona.Name).Validate(requiredText("persona name")),
		huh.NewInput().Title("github host").Prompt("").Value(&editor.persona.Host).Validate(requiredText("github host")),
		huh.NewInput().Title("github account").Prompt("").Value(&editor.persona.Account).Validate(requiredText("github account")),
		huh.NewInput().Title("git author name").Prompt("").Value(&editor.persona.GitName).Validate(requiredText("git author name")),
		huh.NewInput().Title("git author email").Prompt("").Value(&editor.persona.GitEmail).Validate(requiredText("git author email")),
	)
	model.editor = editor
	return model, editor.form.Init()
}

func (model Model) updateEditor(message tea.Msg) (tea.Model, tea.Cmd) {
	if model.editor == nil {
		return model, nil
	}
	if size, ok := message.(tea.WindowSizeMsg); ok {
		model.width = size.Width
		model.height = size.Height
		model.editor.form.WithWidth(model.editorWidth()).WithHeight(model.editorHeight())
	}
	_, cmd := model.editor.form.Update(message)
	switch model.editor.form.State {
	case huh.StateAborted:
		model.editor = nil
		return model, nil
	case huh.StateCompleted:
		switch model.editor.kind {
		case editorProcess:
			action := model.editor.processAction
			model.editor = nil
			return model.beginAction(action)
		case editorPersona:
			if err := model.editor.persona.Validate(); err != nil {
				model.notice = err.Error()
				return model.openPersonaEditor()
			}
			model.confirmation = &confirmationState{kind: editorPersona}
		case editorProject:
			if err := validateEditedProject(model.identities.State, model.editor.project); err != nil {
				model.notice = err.Error()
				return model.openProjectEditor()
			}
			model.confirmation = &confirmationState{kind: editorProject}
		}
		return model, nil
	default:
		return model, cmd
	}
}

func (model Model) updateConfirmation(key string) (tea.Model, tea.Cmd) {
	if model.confirmation == nil {
		return model, nil
	}
	switch key {
	case "esc", "ctrl+c", "n":
		model.confirmation = nil
		model.editor = nil
		return model, nil
	case "left", "right", "tab", "shift+tab":
		model.confirmation.approve = !model.confirmation.approve
		return model, nil
	case "y":
		model.confirmation.approve = true
		return model, nil
	case "enter":
		if !model.confirmation.approve {
			model.confirmation = nil
			model.editor = nil
			return model, nil
		}
		kind := model.confirmation.kind
		model.confirmation = nil
		model.identityLoading = true
		model.savingIdentity = true
		return model, model.saveIdentityCmd(kind)
	default:
		return model, nil
	}
}

func (model Model) renderEditorWorkspace() string {
	p := model.presentation
	width, height := model.viewportSize()
	layout := model.workspaceLayout()
	bodyHeight := max(1, height-headerHeight-shortcutBarHeight)
	header := p.renderHeader(model, width)
	navigator := model.renderEditorNavigator(layout)
	panelWidth := layout.summaryWidth
	panelHeight := bodyHeight
	if !layout.wide {
		panelWidth = width
		panelHeight = max(1, bodyHeight-layout.navigatorHeight)
	}
	var panel string
	if model.confirmation != nil {
		panel = p.renderConfirmation(model, panelWidth, panelHeight)
	} else if model.savingIdentity {
		panel = lipgloss.Place(panelWidth, panelHeight, lipgloss.Center, lipgloss.Center, p.attention.Padding(1, 3).Render("saving..."))
	} else {
		content := ""
		if model.editor != nil && model.editor.form != nil {
			content = p.section.Render(model.editorTitle()) + "\n\n" + model.editor.form.View()
		}
		panel = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(terminalstyle.AccentColor)).
			Padding(1, 2, 0).
			Width(max(1, panelWidth-2)).
			Height(max(1, panelHeight-2)).
			Render(content)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, navigator, panel)
	if !layout.wide {
		body = lipgloss.JoinVertical(lipgloss.Left, navigator, panel)
	}
	footer := model.renderEditorFooter(width)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (model Model) renderEditorNavigator(layout workspaceLayout) string {
	p := model.presentation.dimmed()
	switch model.activeWorkspace {
	case workspaceProjects:
		return p.renderIdentityNavigator(model.identities.State.Projects, model.identitySelected, layout.navigatorWidth, layout.navigatorHeight, layout.wide)
	case workspacePersonas:
		return p.renderPersonaNavigator(model.identities.State.Personas, model.personaSelected, layout.navigatorWidth, layout.navigatorHeight, layout.wide)
	default:
		return p.renderNavigator(model, layout.navigatorWidth, layout.navigatorHeight)
	}
}

func (model Model) editorTitle() string {
	if model.editor == nil {
		return ""
	}
	switch model.editor.kind {
	case editorProcess:
		return "manage process"
	case editorProject:
		return "edit project"
	case editorPersona:
		return "edit persona"
	default:
		return "edit"
	}
}

func (model Model) renderEditorFooter(width int) string {
	p := model.presentation
	if model.confirmation != nil {
		return p.renderConfirmationFooter(width)
	}
	hints := []string{}
	if model.editor != nil && model.editor.kind == editorProcess {
		hints = append(hints, p.renderFooterKey("↑ ↓", "move"), p.renderFooterKey("enter", "select"))
	} else {
		hints = append(hints, p.renderFooterKey("tab", "next field"), p.renderFooterKey("shift+tab", "previous field"), p.renderFooterKey("enter", "continue"))
	}
	hints = append(hints, p.renderFooterKey("esc", "cancel"))
	return p.renderFooter(width, hints)
}

func (p presentation) renderConfirmation(model Model, width, height int) string {
	title := "confirm changes"
	details := "save this configuration?"
	if model.confirmation.kind == editorProject && model.editor != nil {
		details = fmt.Sprintf("save project %s with persona %s?", model.editor.project.Name, model.editor.project.Persona)
	}
	if model.confirmation.kind == editorPersona && model.editor != nil {
		details = fmt.Sprintf("save persona %s (@%s)?", model.editor.persona.Name, model.editor.persona.Account)
	}
	no := p.selected.Padding(0, 2).Render("cancel")
	yes := p.muted.Padding(0, 2).Render("save")
	if model.confirmation.approve {
		no = p.muted.Padding(0, 2).Render("cancel")
		yes = p.success.Padding(0, 2).Render("save")
	}
	modal := p.border.BorderStyle(lipgloss.RoundedBorder()).Padding(1, 3).Width(min(64, max(32, width-8))).Render(
		p.section.Render(title) + "\n\n" + details + "\n\n" + no + "  " + yes,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

func (p presentation) renderConfirmationFooter(width int) string {
	hints := []string{
		p.renderFooterKey("←", "previous"),
		p.renderFooterKey("→", "next"),
		p.renderFooterKey("enter", "confirm"),
		p.renderFooterKey("esc", "cancel"),
	}
	return p.renderFooter(width, hints)
}

func (model Model) saveIdentityCmd(kind editorKind) tea.Cmd {
	editor := model.editor
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(model.ctx, requestTimeout)
		defer cancel()
		var (
			snapshot identity.Snapshot
			err      error
		)
		switch kind {
		case editorProject:
			snapshot, err = model.client.Identity(ctx, identity.Request{Action: "project.put", Project: &editor.project})
		case editorPersona:
			snapshot, err = model.client.Identity(ctx, identity.Request{Action: "persona.put", Persona: &editor.persona})
		}
		return identitySavedMsg{kind: kind, snapshot: snapshot, err: err}
	}
}

func (model Model) editorWidth() int {
	width, _ := model.viewportSize()
	layout := model.workspaceLayout()
	if layout.wide {
		width = layout.summaryWidth
	}
	return max(24, width-4)
}

func (model Model) editorHeight() int {
	_, height := model.viewportSize()
	bodyHeight := height - headerHeight - shortcutBarHeight
	layout := model.workspaceLayout()
	if !layout.wide {
		bodyHeight -= layout.navigatorHeight
	}
	return max(8, bodyHeight-9)
}

func requiredText(label string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}

func validateRepository(value string) error {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("use OWNER/REPO")
	}
	return nil
}

func validateEditedProject(state identity.State, project identity.Project) error {
	copyState := state
	copyState.Projects = append([]identity.Project(nil), state.Projects...)
	for index := range copyState.Projects {
		if copyState.Projects[index].ID == project.ID {
			copyState.Projects[index] = project
		}
	}
	return copyState.Validate()
}
