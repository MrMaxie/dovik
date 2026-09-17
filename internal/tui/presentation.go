package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/supervision"
	"github.com/MrMaxie/dovik/internal/terminalstyle"
	"github.com/charmbracelet/x/ansi"
)

const (
	wideWidth              = 96
	wideHeight             = 24
	headerHeight           = 1
	shortcutBarHeight      = 1
	processActionBarHeight = 1
	truncatedOutputMessage = "earlier lines were discarded; showing the newest output"
)

type workspaceLayout struct {
	wide            bool
	bodyHeight      int
	navigatorWidth  int
	navigatorHeight int
	summaryWidth    int
	summaryHeight   int
	outputWidth     int
	outputHeight    int
}

type presentation struct {
	colorEnabled   bool
	header         lipgloss.Style
	brand          lipgloss.Style
	muted          lipgloss.Style
	section        lipgloss.Style
	project        lipgloss.Style
	selected       lipgloss.Style
	primary        lipgloss.Style
	command        lipgloss.Style
	success        lipgloss.Style
	attention      lipgloss.Style
	danger         lipgloss.Style
	stateMuted     lipgloss.Style
	stdout         lipgloss.Style
	stderr         lipgloss.Style
	key            lipgloss.Style
	border         lipgloss.Style
	fieldLabel     lipgloss.Style
	fieldValue     lipgloss.Style
	footerText     lipgloss.Style
	footerKey      lipgloss.Style
	footerDot      lipgloss.Style
	canvasColor    string
	surfaceColor   string
	primaryColor   string
	mutedColor     string
	successColor   string
	attentionColor string
	dangerColor    string
}

func newPresentation(colorEnabled bool) presentation {
	const (
		canvas    = terminalstyle.CanvasColor
		surface   = terminalstyle.SurfaceColor
		primary   = terminalstyle.PrimaryColor
		muted     = terminalstyle.MutedColor
		accent    = terminalstyle.AccentColor
		success   = terminalstyle.SuccessColor
		attention = terminalstyle.AttentionColor
		danger    = terminalstyle.DangerColor
		border    = terminalstyle.BorderColor
	)

	p := presentation{
		colorEnabled:   colorEnabled,
		canvasColor:    canvas,
		surfaceColor:   surface,
		primaryColor:   primary,
		mutedColor:     muted,
		successColor:   success,
		attentionColor: attention,
		dangerColor:    danger,
		header:         lipgloss.NewStyle(),
		brand:          lipgloss.NewStyle(),
		muted:          lipgloss.NewStyle(),
		section:        lipgloss.NewStyle(),
		project:        lipgloss.NewStyle(),
		selected:       lipgloss.NewStyle(),
		primary:        lipgloss.NewStyle(),
		command:        lipgloss.NewStyle(),
		success:        lipgloss.NewStyle(),
		attention:      lipgloss.NewStyle(),
		danger:         lipgloss.NewStyle(),
		stateMuted:     lipgloss.NewStyle(),
		stdout:         lipgloss.NewStyle(),
		stderr:         lipgloss.NewStyle(),
		key:            lipgloss.NewStyle(),
		border:         lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()),
		fieldLabel:     lipgloss.NewStyle(),
		fieldValue:     lipgloss.NewStyle(),
		footerText:     lipgloss.NewStyle(),
		footerKey:      lipgloss.NewStyle(),
		footerDot:      lipgloss.NewStyle(),
	}
	if !colorEnabled {
		return p
	}

	darkText := lipgloss.Color(canvas)
	p.header = p.header.Foreground(lipgloss.Color(primary)).Background(lipgloss.Color(surface))
	p.brand = p.brand.Bold(true).Foreground(lipgloss.Color(accent))
	p.muted = p.muted.Foreground(lipgloss.Color(muted))
	p.section = p.section.Bold(true).Foreground(lipgloss.Color(primary))
	p.project = p.project.Foreground(lipgloss.Color(muted))
	p.selected = p.selected.Bold(true).Foreground(darkText).Background(lipgloss.Color(accent))
	p.primary = p.primary.Foreground(lipgloss.Color(primary))
	p.command = p.command.Foreground(lipgloss.Color(muted))
	p.success = p.success.Bold(true).Foreground(darkText).Background(lipgloss.Color(success))
	p.attention = p.attention.Bold(true).Foreground(darkText).Background(lipgloss.Color(attention))
	p.danger = p.danger.Bold(true).Foreground(darkText).Background(lipgloss.Color(danger))
	p.stateMuted = p.stateMuted.Bold(true).Foreground(lipgloss.Color(primary)).Background(lipgloss.Color(border))
	p.stdout = p.stdout.Foreground(lipgloss.Color(primary))
	p.stderr = p.stderr.Foreground(lipgloss.Color(danger))
	p.key = p.key.Bold(true).Foreground(lipgloss.Color(accent))
	p.border = p.border.BorderForeground(lipgloss.Color(border)).Foreground(lipgloss.Color(primary))
	p.fieldLabel = p.fieldLabel.Foreground(lipgloss.Color(muted))
	p.fieldValue = p.fieldValue.Foreground(lipgloss.Color(primary))
	p.footerText = p.footerText.Foreground(lipgloss.Color(muted)).Background(lipgloss.Color(surface))
	p.footerKey = p.footerKey.Bold(true).Foreground(lipgloss.Color(accent)).Background(lipgloss.Color(surface))
	p.footerDot = p.footerDot.Foreground(lipgloss.Color(border)).Background(lipgloss.Color(surface))
	return p
}

func (p presentation) dimmed() presentation {
	if !p.colorEnabled {
		return p
	}
	muted := lipgloss.Color(p.mutedColor)
	surface := lipgloss.Color(p.surfaceColor)
	p.section = p.section.Foreground(muted).Faint(true)
	p.project = p.project.Foreground(muted).Faint(true)
	p.selected = p.selected.Foreground(muted).Background(surface).Faint(true)
	p.primary = p.primary.Foreground(muted).Faint(true)
	p.command = p.command.Foreground(muted).Faint(true)
	p.success = p.success.Foreground(muted).Background(surface).Faint(true)
	p.attention = p.attention.Foreground(muted).Background(surface).Faint(true)
	p.danger = p.danger.Foreground(muted).Background(surface).Faint(true)
	p.stateMuted = p.stateMuted.Foreground(muted).Background(surface).Faint(true)
	p.border = p.border.BorderForeground(lipgloss.Color(terminalstyle.BorderColor)).Foreground(muted).Faint(true)
	return p
}

func colorEnabledFromEnvironment() bool {
	value, present := os.LookupEnv("NO_COLOR")
	return !present || value == ""
}

func (model Model) workspaceLayout() workspaceLayout {
	return model.workspaceLayoutWithFooter(shortcutBarHeight)
}

func (model Model) processWorkspaceLayout() workspaceLayout {
	footerHeight := shortcutBarHeight
	if len(model.items) > 0 && model.registry != registryUnavailable {
		footerHeight += processActionBarHeight
	}
	return model.workspaceLayoutWithFooter(footerHeight)
}

func (model Model) workspaceLayoutWithFooter(footerHeight int) workspaceLayout {
	width, height := model.viewportSize()
	bodyHeight := max(1, height-headerHeight-footerHeight)
	if width >= wideWidth && height >= wideHeight {
		navigatorWidth := max(28, min(36, width/3))
		summaryHeight := min(12, max(11, bodyHeight/3))
		return workspaceLayout{
			wide:            true,
			bodyHeight:      bodyHeight,
			navigatorWidth:  navigatorWidth,
			navigatorHeight: bodyHeight,
			summaryWidth:    width - navigatorWidth,
			summaryHeight:   summaryHeight,
			outputWidth:     width - navigatorWidth,
			outputHeight:    bodyHeight - summaryHeight,
		}
	}

	navigatorHeight := max(4, min(7, height/4))
	summaryHeight := min(10, max(9, bodyHeight/3))
	return workspaceLayout{
		bodyHeight:      bodyHeight,
		navigatorWidth:  width,
		navigatorHeight: navigatorHeight,
		summaryWidth:    width,
		summaryHeight:   summaryHeight,
		outputWidth:     width,
		outputHeight:    max(2, bodyHeight-navigatorHeight-summaryHeight),
	}
}

func (model Model) renderProcessWorkspace() string {
	p := model.presentation
	width, _ := model.viewportSize()
	layout := model.processWorkspaceLayout()
	header := p.renderHeader(model, width)
	if model.showOutput && len(model.items) > 0 {
		body := p.renderFullOutput(model, width, layout.bodyHeight)
		footer := p.renderOutputShortcutBar(model, width)
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}
	navigator := p.renderNavigator(model, layout.navigatorWidth, layout.navigatorHeight)
	bodyHeight := layout.bodyHeight
	if !layout.wide {
		bodyHeight = max(1, layout.bodyHeight-layout.navigatorHeight)
	}
	workArea := p.renderRegistryState(model, layout.summaryWidth, bodyHeight)
	if len(model.items) > 0 {
		summary := p.renderSummary(model, layout.summaryWidth, layout.summaryHeight)
		output := p.renderWorkArea(model, layout.outputWidth, layout.outputHeight)
		workArea = lipgloss.JoinVertical(lipgloss.Left, summary, output)
	}
	body := workArea
	if layout.wide {
		body = lipgloss.JoinHorizontal(lipgloss.Top, navigator, workArea)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, navigator, workArea)
	}
	footerRows := []string{}
	if len(model.items) > 0 && model.registry != registryUnavailable {
		footerRows = append(footerRows, p.renderProcessActionBar(model, width))
	}
	footerRows = append(footerRows, p.renderShortcutBar(model, width))
	return lipgloss.JoinVertical(lipgloss.Left, header, body, strings.Join(footerRows, "\n"))
}

func (p presentation) renderFullOutput(model Model, width, height int) string {
	item := model.items[model.selected]
	mode := "paused"
	if model.followOutput {
		mode = "following"
	}
	lines := []string{
		p.renderInlineField("process", string(item.key.processID)),
		p.renderInlineField("project", string(item.key.projectID)),
		p.renderInlineField("viewer", mode),
		p.muted.Render("retained output - no shell"),
		"",
	}
	if model.truncated {
		lines = append(lines, p.muted.Render(truncatedOutputMessage))
	}
	output := model.outputLines()
	pageHeight := max(1, height-len(lines)-1)
	end := max(0, len(output)-model.scroll)
	start := max(0, end-pageHeight)
	for _, line := range output[start:end] {
		prefix := p.stdout.Render("[stdout]")
		if line.stream == supervision.OutputStreamStderr {
			prefix = p.stderr.Render("[stderr]")
		}
		lines = append(lines, prefix+" "+terminalstyle.RenderOutput(line.text, p.colorEnabled))
	}
	if len(output) == 0 {
		lines = append(lines, p.muted.Render("no output yet."))
	}
	return p.border.BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		Padding(1, 2, 0).Width(width).Height(height).Render(p.fitLines(lines, max(1, width-4), max(1, height-1)))
}

func (p presentation) renderOutputShortcutBar(model Model, width int) string {
	follow := "follow"
	if model.followOutput {
		follow = "pause"
	}
	hints := []string{
		p.renderFooterKey("esc", "close"),
		p.renderFooterKey("f", follow),
		p.renderFooterKey("↑ ↓", "scroll"),
		p.renderFooterKey("pgup pgdn", "page"),
		p.renderFooterKey("home", "top"),
		p.renderFooterKey("end", "bottom"),
	}
	return p.renderFooter(width, hints)
}

func (model Model) viewportSize() (int, int) {
	width, height := model.width, model.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	return width, height
}

func (p presentation) renderHeader(model Model, width int) string {
	brandStyle := p.brand
	if p.colorEnabled {
		background := lipgloss.Color(p.surfaceColor)
		brandStyle = brandStyle.Background(background)
	}
	brand := brandStyle.Render(" dovik ")
	tabs := p.renderWorkspaceTabs(model)
	status := p.renderDaemonIndicator(model)
	leftWidth := max(0, width-lipgloss.Width(status)-1)
	left := ansi.Truncate(brand+" "+tabs, leftWidth, "")
	spacing := p.header.Render(strings.Repeat(" ", max(1, width-lipgloss.Width(left)-lipgloss.Width(status))))
	return p.header.Width(width).MaxWidth(width).Render(ansi.Truncate(left+spacing+status, width, ""))
}

func (p presentation) renderWorkspaceTabs(model Model) string {
	tabs := []struct {
		label string
		value workspaceTab
	}{
		{label: fmt.Sprintf("personas (%d)", len(model.identities.State.Personas)), value: workspacePersonas},
		{label: fmt.Sprintf("projects (%d)", len(model.identities.State.Projects)), value: workspaceProjects},
		{label: fmt.Sprintf("processes (%d/%d)", model.runningProcessCount(), len(model.items)), value: workspaceProcesses},
	}
	rendered := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		label := " " + tab.label + " "
		if tab.value == model.activeWorkspace {
			if p.colorEnabled {
				rendered = append(rendered, p.selected.Render(label))
			} else {
				rendered = append(rendered, "["+tab.label+"]")
			}
			continue
		}
		style := p.muted
		if p.colorEnabled {
			style = style.Background(lipgloss.Color(p.surfaceColor))
		}
		rendered = append(rendered, style.Render(label))
	}
	return strings.Join(rendered, " ")
}

func (p presentation) renderDaemonIndicator(model Model) string {
	label := "daemon checking"
	dotColor := p.mutedColor
	if model.daemonStarting {
		label = "daemon starting"
		dotColor = p.attentionColor
	} else {
		switch model.registry {
		case registryEmpty, registryPopulated:
			label = "daemon online"
			dotColor = p.successColor
		case registryUnavailable:
			label = "daemon offline"
			dotColor = p.dangerColor
		}
	}
	if !p.colorEnabled {
		return "• " + label
	}
	background := lipgloss.Color(p.surfaceColor)
	dot := lipgloss.NewStyle().Foreground(lipgloss.Color(dotColor)).Background(background).Render("•")
	text := lipgloss.NewStyle().Foreground(lipgloss.Color(p.primaryColor)).Background(background).Render(" " + label)
	return dot + text
}

func (p presentation) renderNavigator(model Model, width, height int) string {
	capacity := max(1, height-1)
	lines := []string{}
	switch {
	case model.loading:
		lines = append(lines, p.muted.Render("loading registered processes..."))
	case model.registry == registryUnavailable:
		lines = append(lines, p.muted.Render("process list unavailable."))
	case len(model.items) == 0:
		lines = append(lines, p.muted.Render("no process definitions."))
	default:
		lines = p.renderProcessTree(model, width-2, capacity)
	}
	style := p.border.BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(model.processWorkspaceLayout().wide)
	return style.Width(width).Height(height).Render(p.fitLines(lines, width-2, height))
}

func (model Model) runningProcessCount() int {
	running := 0
	for _, item := range model.items {
		if item.known && item.state == supervision.ProcessStateRunning {
			running++
		}
	}
	return running
}

func (p presentation) renderProcessTree(model Model, width, capacity int) []string {
	start, end := model.navigatorRange(max(1, capacity/2))
	projects := map[supervision.ProjectID][]int{}
	order := make([]supervision.ProjectID, 0)
	for index, item := range model.items {
		if _, exists := projects[item.key.projectID]; !exists {
			order = append(order, item.key.projectID)
		}
		projects[item.key.projectID] = append(projects[item.key.projectID], index)
	}
	lines := make([]string, 0, capacity)
	for _, projectID := range order {
		indices := projects[projectID]
		visible := false
		for _, index := range indices {
			if index >= start && index < end {
				visible = true
				break
			}
		}
		if !visible {
			continue
		}
		lines = append(lines, p.section.Render(" "+string(projectID)))
		for childIndex, index := range indices {
			if index < start || index >= end {
				continue
			}
			branch := "├─"
			if childIndex == len(indices)-1 {
				branch = "└─"
			}
			lines = append(lines, p.renderProcessRow(model.items[index], branch, index == model.selected, width))
			if len(lines) >= capacity {
				return lines
			}
		}
	}
	return lines
}

func (p presentation) renderProcessRow(item processItem, branch string, selected bool, width int) string {
	selector := " "
	style := p.project
	if selected {
		selector = "›"
		style = p.selected
	}
	prefix := selector + " " + branch + " "
	marker := "○"
	markerStyle := style
	if item.known && item.state.IsActive() {
		marker = "●"
		if p.colorEnabled {
			markerStyle = markerStyle.Foreground(lipgloss.Color(p.successColor))
		}
	}
	suffix := " " + string(item.key.processID)
	if !selected {
		return ansi.Truncate(style.Render(prefix)+markerStyle.Render(marker)+style.Render(suffix), width, "...")
	}

	suffixWidth := max(0, width-lipgloss.Width(prefix)-lipgloss.Width(marker))
	suffix = ansi.Truncate(suffix, suffixWidth, "")
	return style.Render(prefix) + markerStyle.Render(marker) + style.Width(suffixWidth).MaxWidth(suffixWidth).Render(suffix)
}

func (p presentation) renderSummary(model Model, width, height int) string {
	lines := []string{}

	item := model.items[model.selected]
	state := "loading"
	if model.statusKnown && model.registry != registryUnavailable {
		state = string(model.currentState())
	}
	if model.registry == registryUnavailable || (!model.statusKnown && model.diagnostic != "") {
		state = "unavailable"
	}
	lines = append(lines,
		p.renderField("process", string(item.key.processID)),
		p.renderField("project", string(item.key.projectID)),
		p.fieldLabel.Render("status"),
		"  "+p.renderState(state),
		p.renderField("command", item.command),
	)
	if model.loading {
		lines = append(lines, p.muted.Render("refreshing process list..."))
	} else if model.notice != "" {
		style := p.attention
		if model.diagnostic != "" {
			style = p.danger
		}
		lines = append(lines, style.Render(model.notice))
	}
	return lipgloss.NewStyle().Padding(1, 1, 0).Width(width).Height(height).Render(p.fitLines(lines, width-2, height-1))
}

func (p presentation) renderProcessActionBar(model Model, width int) string {
	actions := []string{}
	if model.pending {
		actions = append(actions, p.attention.Render("working..."))
		return p.renderActionFooter(width, actions)
	}
	if !model.statusKnown || model.registry == registryUnavailable || model.loading {
		return p.renderActionFooter(width, actions)
	}
	if model.currentState().IsActive() {
		actions = append(actions, p.renderFooterKey("x", "stop"), p.renderFooterKey("r", "restart"))
		return p.renderActionFooter(width, actions)
	}
	actions = append(actions, p.renderFooterKey("s", "start"))
	return p.renderActionFooter(width, actions)
}

func (p presentation) renderKey(key, action string) string {
	return p.key.Render(key) + " " + p.muted.Render(strings.ToLower(action))
}

func (p presentation) hintSeparator() string {
	if !p.colorEnabled {
		return "  •  "
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(terminalstyle.BorderColor)).Render("  •  ")
}

func (p presentation) renderField(label, value string) string {
	return p.fieldLabel.Render(label) + "\n" + p.fieldValue.Render("  "+value)
}

func (p presentation) renderFooterKey(key, action string) string {
	if !p.colorEnabled {
		return key + " " + strings.ToLower(action)
	}
	return p.footerKey.Render(key) + p.footerText.Render(" "+strings.ToLower(action))
}

func (p presentation) footerSeparator() string {
	if !p.colorEnabled {
		return " • "
	}
	return p.footerDot.Render(" • ")
}

func (p presentation) renderFooter(width int, hints []string) string {
	content := " " + strings.Join(hints, p.footerSeparator())
	return p.header.Width(width).MaxWidth(width).Render(ansi.Truncate(content, width, ""))
}

func (p presentation) renderActionFooter(width int, actions []string) string {
	content := " " + strings.Join(actions, p.footerSeparator())
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(ansi.Truncate(content, max(1, width), ""))
}

func (p presentation) renderState(state string) string {
	label := strings.ToUpper(state)
	if !p.colorEnabled {
		return "[" + label + "]"
	}
	style := p.stateMuted
	switch supervision.ProcessState(state) {
	case supervision.ProcessStateRunning:
		style = p.success
	case supervision.ProcessStateStarting, supervision.ProcessStateStopping:
		style = p.attention
	case supervision.ProcessStateFailed:
		style = p.danger
	}
	return style.Padding(0, 1).Render(label)
}

func (p presentation) renderWorkArea(model Model, width, height int) string {
	if model.showHelp {
		return p.panel(width, height, p.helpLines())
	}
	if model.showDetail {
		return p.panel(width, height, p.detailLines(model))
	}
	return p.panel(width, height, p.outputLines(model, max(1, height-1)))
}

func (p presentation) outputLines(model Model, height int) []string {
	lines := []string{p.renderOutputHeading()}
	if model.truncated {
		lines = append(lines, p.muted.Render(truncatedOutputMessage))
	}
	visible := model.visibleOutputLines(max(1, height-len(lines)))
	if len(visible) == 0 {
		lines = append(lines, p.muted.Render("no output yet."))
		return lines
	}
	for _, line := range visible {
		prefix := p.stdout.Render("[stdout]")
		if line.stream == supervision.OutputStreamStderr {
			prefix = p.stderr.Render("[stderr]")
		}
		lines = append(lines, prefix+" "+terminalstyle.RenderOutput(line.text, p.colorEnabled))
	}
	return lines
}

func (p presentation) renderOutputHeading() string {
	if !p.colorEnabled {
		return "process output"
	}
	return p.section.Foreground(lipgloss.Color(terminalstyle.AccentColor)).Render("process output")
}

func (p presentation) detailLines(model Model) []string {
	lines := []string{p.section.Render("runtime details")}
	if model.diagnostic != "" {
		lines = append(lines, p.danger.Render("diagnostic  ")+model.diagnostic)
	}
	if !model.hasRuntime {
		return append(lines, p.muted.Render("no runtime has started."))
	}
	pid := "-"
	if model.runtime.PID != nil {
		pid = fmt.Sprint(*model.runtime.PID)
	}
	exitCode := "-"
	if model.runtime.ExitCode != nil {
		exitCode = fmt.Sprint(*model.runtime.ExitCode)
	}
	lines = append(lines,
		fmt.Sprintf("instance  %s    pid  %s", model.runtime.InstanceID, pid),
		fmt.Sprintf("started   %s", formatTime(model.runtime.StartedAt)),
		fmt.Sprintf("ended     %s    exit  %s    reason  %s", formatTime(model.runtime.FinishedAt), exitCode, emptyAsDash(model.runtime.TerminationReason)),
	)
	return lines
}

func (p presentation) helpLines() []string {
	return []string{
		p.section.Render("keyboard help"),
		p.renderKey("← →", "switch workspace"),
		p.renderKey("↑ ↓", "move selection"),
		p.renderKey("enter", "manage selected process"),
		p.renderKey("s", "start") + p.hintSeparator() + p.renderKey("x", "stop") + p.hintSeparator() + p.renderKey("r", "restart"),
		p.renderKey("pgup", "page up") + p.hintSeparator() + p.renderKey("pgdn", "page down"),
		p.renderKey("l", "refresh") + p.hintSeparator() + p.renderKey("d", "details") + p.hintSeparator() + p.renderKey("?", "close help") + p.hintSeparator() + p.renderKey("q", "quit"),
	}
}

func (p presentation) renderShortcutBar(model Model, width int) string {
	shortcuts := []string{}
	if len(model.items) > 1 && !model.pending && !model.loading && model.registry != registryUnavailable {
		shortcuts = append(shortcuts, p.renderFooterKey("↑ ↓", "move"))
	}
	if len(model.items) > 0 && !model.pending && !model.loading && model.registry != registryUnavailable {
		shortcuts = append(shortcuts, p.renderFooterKey("enter", "manage"))
	}
	if len(model.items) > 0 && model.registry != registryUnavailable {
		shortcuts = append(shortcuts, p.renderFooterKey("o", "output"))
	}
	if !model.pending && !model.loading && !model.daemonStarting {
		if model.registry == registryUnavailable && model.daemonLauncher != nil {
			shortcuts = append(shortcuts, p.renderFooterKey("s", "start daemon"))
		}
		label := "refresh"
		if model.registry == registryUnavailable || (!model.statusKnown && model.diagnostic != "") {
			label = "retry"
		}
		shortcuts = append(shortcuts, p.renderFooterKey("l", label))
	}
	shortcuts = append(shortcuts, p.renderFooterKey("← →", "tabs"))
	if model.showDetail {
		shortcuts = append(shortcuts, p.renderFooterKey("d", "close details"))
	} else if model.diagnostic != "" {
		shortcuts = append(shortcuts, p.renderFooterKey("d", "details"))
	}
	if model.showHelp {
		shortcuts = append(shortcuts, p.renderFooterKey("?", "close help"))
	} else if model.registry == registryEmpty {
		shortcuts = append(shortcuts, p.renderFooterKey("?", "setup help"))
	} else if width >= 120 {
		shortcuts = append(shortcuts, p.renderFooterKey("?", "help"))
	}
	shortcuts = append(shortcuts, p.renderFooterKey("q", "quit"))
	return p.renderFooter(width, shortcuts)
}

func (p presentation) fitLines(lines []string, width, height int) string {
	fitted := []string{}
	for _, line := range lines {
		for _, wrapped := range strings.Split(ansi.Hardwrap(line, max(1, width), true), "\n") {
			if len(fitted) == height {
				return strings.Join(fitted, "\n")
			}
			fitted = append(fitted, wrapped)
		}
	}
	return strings.Join(fitted, "\n")
}

func (p presentation) panel(width, height int, lines []string) string {
	for index, line := range lines {
		lines[index] = ansi.Truncate(line, width-2, "")
	}
	return p.border.BorderTop(true).BorderBottom(false).BorderLeft(false).BorderRight(false).
		Padding(0, 1).Width(width).Height(height).Render(p.fitLines(lines, width-2, height-1))
}

func (p presentation) renderRegistryState(model Model, width, height int) string {
	lines := []string{}
	switch model.registry {
	case registryLoading:
		lines = append(lines, p.section.Render("loading processes..."), "", p.muted.Render("waiting for the daemon."))
	case registryUnavailable:
		lines = append(lines, p.section.Render("process data is unavailable"), "")
		switch {
		case model.daemonStarting:
			lines = append(lines, p.attention.Render("starting daemon..."))
		case model.notice != "":
			lines = append(lines, p.danger.Render(model.notice))
		case model.daemonLauncher != nil:
			lines = append(lines, "Press s to start the daemon, or l to retry the connection.")
		default:
			lines = append(lines, "Check that the daemon is running, then press l to retry.")
		}
	case registryEmpty:
		lines = append(lines, p.section.Render("no processes registered"), "", "register a project and process using the CLI.", "press ? for setup commands, then l to refresh.")
	}
	if model.showHelp {
		if model.registry == registryEmpty {
			lines = p.setupLines()
		} else {
			lines = append(lines, "")
			if model.daemonLauncher != nil && !model.daemonStarting {
				lines = append(lines, p.renderKey("s", "Start daemon"))
			}
			lines = append(lines, p.renderKey("l", "Retry when ready"), p.renderKey("?", "Close help"), p.renderKey("q", "Quit"))
		}
	} else if model.showDetail && model.diagnostic != "" {
		lines = append(lines, "", p.section.Render("Diagnostic details"), model.diagnostic)
	}
	return lipgloss.NewStyle().Padding(2, 2, 0).Width(width).Height(height).Render(p.fitLines(lines, width-4, height-2))
}

func (p presentation) setupLines() []string {
	return []string{
		p.section.Render("register your first process"), "",
		"run these commands in another terminal:",
		`dovik project add --id PROJECT --root "ROOT"`,
		`dovik process add --project PROJECT --id PROCESS`,
		`  --command "COMMAND"`,
		"the process command above is one line.", "",
		"replace PROJECT and PROCESS with your chosen IDs.",
		"ROOT is an absolute folder; COMMAND is an executable.",
		"add --arg VALUE for each command argument.", "",
		"return here, press ? to close help, then l to refresh.",
	}
}

func (model Model) renderSmallTerminal() string {
	lines := []string{
		"DOVIK",
		"",
		fmt.Sprintf("Terminal too small: need at least %dx%d, current %dx%d.", minimumWidth, minimumHeight, model.width, model.height),
		"",
	}
	if model.showHelp {
		lines = append(lines, "? close help", "q quit")
	} else {
		lines = append(lines, "? help  q quit")
	}
	return strings.Join(lines, "\n")
}
