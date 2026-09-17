package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/charmbracelet/x/ansi"
)

type identityLoadedMsg struct {
	snapshot identity.Snapshot
	err      error
}

func (model Model) loadIdentityCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(model.ctx, requestTimeout)
		defer cancel()
		snapshot, err := model.client.Identity(ctx, identity.Request{Action: "get"})
		return identityLoadedMsg{snapshot, err}
	}
}

func (model Model) identityView() string {
	p := model.presentation
	width, height := model.viewportSize()
	bodyHeight := max(1, height-headerHeight-shortcutBarHeight)
	header := p.renderHeader(model, width)
	footer := p.renderIdentityShortcutBar(model, width)
	projects := model.identities.State.Projects
	if model.identityLoading {
		body := p.renderIdentityState(width, bodyHeight, "loading projects", "reading project identity and session state...")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}
	if model.identityError != "" {
		body := p.renderIdentityState(width, bodyHeight, "projects unavailable", "press l to retry. process management remains available from the processes tab.")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}
	if len(projects) == 0 {
		body := p.renderIdentityState(width, bodyHeight, "no projects configured", "GitHub CLI works normally. configure a repository when you want to assign a project identity.")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}

	selected := min(model.identitySelected, len(projects)-1)
	layout := model.workspaceLayout()
	navigator := p.renderIdentityNavigator(projects, selected, layout.navigatorWidth, layout.navigatorHeight, layout.wide)
	detailsHeight := layout.bodyHeight
	if !layout.wide {
		detailsHeight = max(1, layout.bodyHeight-layout.navigatorHeight)
	}
	details := p.renderIdentityDetails(model, projects[selected], layout.summaryWidth, detailsHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, navigator, details)
	if !layout.wide {
		body = lipgloss.JoinVertical(lipgloss.Left, navigator, details)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (model Model) personaView() string {
	p := model.presentation
	width, height := model.viewportSize()
	bodyHeight := max(1, height-headerHeight-shortcutBarHeight)
	header := p.renderHeader(model, width)
	footer := p.renderPersonaShortcutBar(model, width)
	personas := model.identities.State.Personas
	if model.identityLoading {
		body := p.renderIdentityState(width, bodyHeight, "loading personas", "reading saved personas and project assignments...")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}
	if model.identityError != "" {
		body := p.renderIdentityState(width, bodyHeight, "personas unavailable", "press l to retry. processes remain available from the processes tab.")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}
	if len(personas) == 0 {
		body := p.renderIdentityState(width, bodyHeight, "no personas configured", "configure a repository identity to create or assign a persona.")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}

	selected := min(model.personaSelected, len(personas)-1)
	layout := model.workspaceLayout()
	navigator := p.renderPersonaNavigator(personas, selected, layout.navigatorWidth, layout.navigatorHeight, layout.wide)
	detailsHeight := layout.bodyHeight
	if !layout.wide {
		detailsHeight = max(1, layout.bodyHeight-layout.navigatorHeight)
	}
	details := p.renderPersonaDetails(model, personas[selected], layout.summaryWidth, detailsHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, navigator, details)
	if !layout.wide {
		body = lipgloss.JoinVertical(lipgloss.Left, navigator, details)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (p presentation) renderIdentityNavigator(projects []identity.Project, selected, width, height int, wide bool) string {
	lines := []string{}
	capacity := max(1, height)
	start := max(0, selected-capacity/2)
	end := min(len(projects), start+capacity)
	start = max(0, end-capacity)
	for index := start; index < end; index++ {
		label := "  " + projects[index].Name
		if index == selected {
			label = "› " + projects[index].Name
			lines = append(lines, p.selected.Width(max(1, width-2)).MaxWidth(max(1, width-2)).Render(label))
			continue
		}
		lines = append(lines, ansi.Truncate(p.project.Render(label), width-2, "..."))
	}
	style := p.border.BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(wide)
	return style.Width(width).Height(height).Render(p.fitLines(lines, width-2, height))
}

func (p presentation) renderPersonaNavigator(personas []identity.Persona, selected, width, height int, wide bool) string {
	lines := []string{}
	capacity := max(1, height)
	start := max(0, selected-capacity/2)
	end := min(len(personas), start+capacity)
	start = max(0, end-capacity)
	for index := start; index < end; index++ {
		label := "  " + personas[index].Name
		if index == selected {
			label = "› " + personas[index].Name
			lines = append(lines, p.selected.Width(max(1, width-2)).MaxWidth(max(1, width-2)).Render(label))
			continue
		}
		lines = append(lines, ansi.Truncate(p.project.Render(label), width-2, "..."))
	}
	style := p.border.BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(wide)
	return style.Width(width).Height(height).Render(p.fitLines(lines, width-2, height))
}

func (p presentation) renderIdentityDetails(model Model, project identity.Project, width, height int) string {
	persona, _ := model.identities.State.FindPersona(project.Persona)
	personaSwitching := "not allowed"
	if project.Policy.Allows("persona") {
		personaSwitching = "allowed"
	}
	sessions, active := 0, 0
	for _, session := range model.identities.Sessions {
		if session.ProjectID != project.ID {
			continue
		}
		sessions++
		if session.Status == "active" {
			active++
		}
	}

	repository := project.Repository
	if persona.Host != "" {
		repository = persona.Host + "/" + repository
	}
	account := "not assigned"
	if persona.Account != "" {
		account = persona.Name + "  @" + persona.Account
	}
	author := "not configured"
	if persona.GitName != "" || persona.GitEmail != "" {
		author = strings.TrimSpace(persona.GitName + " <" + persona.GitEmail + ">")
	}
	protection := humanizeIdentityLabel(project.Mode)
	policy := humanizeIdentityLabel(project.Policy.Preset)
	sessionSummary := "no isolated sessions"
	if sessions > 0 {
		sessionSummary = fmt.Sprintf("%d active, %d waiting", active, sessions-active)
	}
	boundary := "commands use the assigned identity and policy."
	if project.Mode == "proxy-level" {
		boundary = "same-user applications are outside this boundary."
	}
	if height < 22 {
		return p.renderCompactIdentityDetails(project, repository, account, author, protection, policy, personaSwitching, sessionSummary, boundary, width, height)
	}

	lines := []string{
		p.renderField("project", project.Name),
	}
	if description := strings.Join(strings.Fields(project.Description), " "); description != "" {
		lines = append(lines, p.renderField("description", description))
	}
	lines = append(lines,
		p.renderField("repository", repository),
		p.renderField("persona", account),
		p.renderField("git author", author),
		p.renderField("protection", protection),
		p.renderField("policy", policy),
		p.section.Render("repository operations"),
	)
	lines = append(lines, p.renderOperationGrid(project.Policy, max(1, width-4))...)
	lines = append(lines,
		p.renderField("persona switching", personaSwitching),
		p.renderField("agent sessions", sessionSummary),
		p.muted.Render(boundary),
	)
	return lipgloss.NewStyle().Padding(1, 2, 0).Width(width).Height(height).Render(p.fitLines(lines, max(1, width-4), max(1, height-1)))
}

func (p presentation) renderCompactIdentityDetails(project identity.Project, repository, account, author, protection, policy, personaSwitching, sessionSummary, boundary string, width, height int) string {
	lines := []string{
		p.fieldLabel.Render("project"),
		p.fieldValue.Render("  " + project.Name),
	}
	if description := strings.Join(strings.Fields(project.Description), " "); description != "" {
		lines = append(lines, p.renderInlineField("description", description))
	}
	lines = append(lines,
		p.renderInlineField("repository", repository),
		p.renderInlineField("persona", account),
		p.renderInlineField("git author", author),
		p.fieldLabel.Render("protection"),
		p.fieldValue.Render("  "+protection),
		p.renderInlineField("policy", policy),
		p.section.Render("repository operations"),
	)
	lines = append(lines, p.renderOperationGrid(project.Policy, max(1, width-4))...)
	lines = append(lines,
		p.renderInlineField("persona switching", personaSwitching),
		p.renderInlineField("agent sessions", sessionSummary),
	)
	if len(lines) < height {
		lines = append(lines, p.muted.Render(boundary))
	}
	return lipgloss.NewStyle().Padding(0, 2).Width(width).Height(height).Render(p.fitLines(lines, max(1, width-4), max(1, height)))
}

func (p presentation) renderInlineField(label, value string) string {
	return p.fieldLabel.Render(label) + "  " + p.fieldValue.Render(value)
}

func (p presentation) renderOperationGrid(policy identity.Policy, width int) []string {
	columns := 1
	if width >= 46 {
		columns = 2
	}
	cellWidth := max(1, width/columns)
	items := make([]string, 0, len(identity.RepositoryOperations))
	for _, operation := range identity.RepositoryOperations {
		allowed := policy.Allows(operation)
		marker := p.operationMarker(allowed)
		items = append(items, "  "+marker+" "+p.fieldValue.Render(operation))
	}
	lines := make([]string, 0, (len(items)+columns-1)/columns)
	for index := 0; index < len(items); index += columns {
		line := items[index]
		if columns == 2 && index+1 < len(items) {
			padding := max(2, cellWidth-lipgloss.Width(line))
			line += strings.Repeat(" ", padding) + items[index+1]
		}
		lines = append(lines, line)
	}
	return lines
}

func (p presentation) operationMarker(allowed bool) string {
	marker := "○"
	if !allowed {
		return marker
	}
	marker = "●"
	if !p.colorEnabled {
		return marker
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(p.successColor)).Render(marker)
}

func (p presentation) renderPersonaDetails(model Model, persona identity.Persona, width, height int) string {
	account := "not configured"
	if persona.Account != "" {
		account = "@" + persona.Account
	}
	host := persona.Host
	if host == "" {
		host = "not configured"
	}
	authorName := emptyAsNotConfigured(persona.GitName)
	authorEmail := emptyAsNotConfigured(persona.GitEmail)
	projects := make([]string, 0)
	for _, project := range model.identities.State.Projects {
		if project.Persona == persona.ID {
			projects = append(projects, project.Name)
		}
	}
	if height < 22 {
		return p.renderCompactPersonaDetails(persona, account, host, authorName, authorEmail, projects, width, height)
	}
	lines := []string{
		p.renderField("persona", persona.Name),
		p.renderField("persona id", persona.ID),
		p.renderField("github account", account),
		p.renderField("github host", host),
		p.renderField("git author name", authorName),
		p.renderField("git author email", authorEmail),
		p.section.Render("assigned projects"),
	}
	if len(projects) == 0 {
		lines = append(lines, p.muted.Render("  ○ not assigned to a project"))
	} else {
		for _, project := range projects {
			lines = append(lines, "  "+p.operationMarker(true)+" "+p.fieldValue.Render(project))
		}
	}
	return lipgloss.NewStyle().Padding(1, 2, 0).Width(width).Height(height).Render(p.fitLines(lines, max(1, width-4), max(1, height-1)))
}

func (p presentation) renderCompactPersonaDetails(persona identity.Persona, account, host, authorName, authorEmail string, projects []string, width, height int) string {
	lines := []string{
		p.fieldLabel.Render("persona"),
		p.fieldValue.Render("  " + persona.Name),
		p.renderInlineField("persona id", persona.ID),
		p.renderInlineField("github account", account),
		p.renderInlineField("github host", host),
		p.renderInlineField("git author name", authorName),
		p.renderInlineField("git author email", authorEmail),
		p.section.Render("assigned projects"),
	}
	if len(projects) == 0 {
		lines = append(lines, p.muted.Render("  ○ not assigned to a project"))
	} else {
		available := max(1, height-len(lines))
		visible := min(len(projects), available)
		for _, project := range projects[:visible] {
			lines = append(lines, "  "+p.operationMarker(true)+" "+p.fieldValue.Render(project))
		}
		if visible < len(projects) {
			lines[len(lines)-1] = p.muted.Render(fmt.Sprintf("... %d more", len(projects)-visible+1))
		}
	}
	return lipgloss.NewStyle().Padding(0, 2).Width(width).Height(height).Render(p.fitLines(lines, max(1, width-4), max(1, height)))
}

func emptyAsNotConfigured(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not configured"
	}
	return value
}

func (p presentation) renderIdentityMode(mode string) string {
	label := strings.ToUpper(humanizeIdentityLabel(mode))
	style := p.stateMuted
	if mode == "agent-isolation" {
		style = p.success
	}
	if !p.colorEnabled {
		return "[" + label + "]"
	}
	return style.Padding(0, 1).Render(label)
}

func (p presentation) renderIdentityState(width, height int, title, message string) string {
	lines := []string{p.section.Render(title), "", p.muted.Render(message)}
	return lipgloss.NewStyle().Padding(2, 3).Width(width).Height(height).Render(p.fitLines(lines, max(1, width-6), max(1, height-2)))
}

func (p presentation) renderIdentityShortcutBar(model Model, width int) string {
	shortcuts := []string{}
	if len(model.identities.State.Projects) > 1 && !model.identityLoading {
		shortcuts = append(shortcuts, p.renderFooterKey("↑ ↓", "move"))
	}
	if !model.identityLoading && model.identityError == "" && !model.pending && !model.daemonStarting && model.registry != registryUnavailable {
		shortcuts = append(shortcuts, p.renderFooterKey("enter", "edit"))
	}
	shortcuts = append(shortcuts,
		p.renderFooterKey("l", "refresh"),
		p.renderFooterKey("← →", "tabs"),
		p.renderFooterKey("q", "quit"),
	)
	return p.renderFooter(width, shortcuts)
}

func (p presentation) renderPersonaShortcutBar(model Model, width int) string {
	shortcuts := []string{}
	if len(model.identities.State.Personas) > 1 && !model.identityLoading {
		shortcuts = append(shortcuts, p.renderFooterKey("↑ ↓", "move"))
	}
	if len(model.identities.State.Personas) > 0 && !model.identityLoading && model.identityError == "" {
		shortcuts = append(shortcuts, p.renderFooterKey("enter", "edit"))
	}
	shortcuts = append(shortcuts,
		p.renderFooterKey("l", "refresh"),
		p.renderFooterKey("← →", "tabs"),
		p.renderFooterKey("q", "quit"),
	)
	return p.renderFooter(width, shortcuts)
}

func humanizeIdentityLabel(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	if value == "" {
		return "Not configured"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
