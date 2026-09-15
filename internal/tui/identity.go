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
	header := p.renderHeaderTitle(model, width, "Project identity")
	footer := p.renderIdentityShortcutBar(model, width)
	projects := model.identities.State.Projects
	if model.identityLoading {
		body := p.renderIdentityState(width, bodyHeight, "Loading identity", "Reading project identity and session state...")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}
	if model.identityError != "" {
		body := p.renderIdentityState(width, bodyHeight, "Identity unavailable", "Press i to retry. Process management remains available from the main workspace.")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}
	if len(projects) == 0 {
		body := p.renderIdentityState(width, bodyHeight, "No identity configured", "GitHub CLI works normally. Press c only when you want to assign a project identity and protection policy.")
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

func (p presentation) renderIdentityNavigator(projects []identity.Project, selected, width, height int, wide bool) string {
	lines := []string{p.section.Render("Projects") + p.muted.Render(fmt.Sprintf("  %d/%d", selected+1, len(projects)))}
	capacity := max(1, height-2)
	start := max(0, selected-capacity/2)
	end := min(len(projects), start+capacity)
	start = max(0, end-capacity)
	for index := start; index < end; index++ {
		label := "  " + projects[index].Name
		if index == selected {
			label = "> " + projects[index].Name
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
	allowed := make([]string, 0, len(identity.RepositoryOperations))
	for _, permission := range identity.RepositoryOperations {
		if project.Policy.Allows(permission) {
			allowed = append(allowed, permission)
		}
	}
	personaSwitching := "Not allowed"
	if project.Policy.Allows("persona") {
		personaSwitching = "Allowed"
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
	account := "Not assigned"
	if persona.Account != "" {
		account = persona.Name + "  " + p.muted.Render("@"+persona.Account)
	}
	author := "Not configured"
	if persona.GitName != "" || persona.GitEmail != "" {
		author = strings.TrimSpace(persona.GitName + " <" + persona.GitEmail + ">")
	}
	protection := humanizeIdentityLabel(project.Mode)
	policy := humanizeIdentityLabel(project.Policy.Preset)
	sessionSummary := "No isolated sessions"
	if sessions > 0 {
		sessionSummary = fmt.Sprintf("%d active, %d waiting", active, sessions-active)
	}
	boundary := "Commands use the assigned identity and policy."
	if project.Mode == "proxy-level" {
		boundary = "Same-user applications are outside this boundary."
	}

	lines := []string{
		p.primary.Bold(p.colorEnabled).Render(project.Name) + "  " + p.renderIdentityMode(project.Mode),
		p.muted.Render(repository),
	}
	if description := strings.Join(strings.Fields(project.Description), " "); description != "" {
		lines = append(lines, p.muted.Render(description))
	}
	lines = append(lines,
		"",
		p.section.Render("GitHub account"),
		account,
		"",
		p.section.Render("Git commits"),
		author,
		"",
		p.section.Render("Protection"),
		fmt.Sprintf("%s  %s", protection, p.muted.Render("Policy: "+policy)),
		fmt.Sprintf("%d of %d operations  %s", len(allowed), len(identity.RepositoryOperations), p.muted.Render(strings.Join(allowed, ", "))),
		fmt.Sprintf("Persona switching  %s", p.muted.Render(personaSwitching)),
		p.muted.Render(boundary),
		"",
		p.section.Render("Agent sessions"),
		sessionSummary,
	)
	return lipgloss.NewStyle().Padding(1, 2, 0).Width(width).Height(height).Render(p.fitLines(lines, max(1, width-4), max(1, height-1)))
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
		shortcuts = append(shortcuts, p.renderKey("j/k", "Select"))
	}
	if !model.identityLoading && model.identityError == "" && !model.pending && !model.daemonStarting && model.registry != registryUnavailable {
		shortcuts = append(shortcuts, p.renderKey("c", "Configure"))
	}
	shortcuts = append(shortcuts, p.renderKey("i", "Refresh"), p.renderKey("esc", "Processes"), p.renderKey("q", "Quit"))
	return p.header.Width(width).MaxWidth(width).Render(ansi.Truncate(" "+strings.Join(shortcuts, "  "), width, ""))
}

func humanizeIdentityLabel(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	if value == "" {
		return "Not configured"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
