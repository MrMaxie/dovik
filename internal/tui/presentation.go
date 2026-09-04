package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/supervision"
	"github.com/charmbracelet/x/ansi"
)

const (
	wideWidth         = 96
	wideHeight        = 24
	headerHeight      = 1
	shortcutBarHeight = 1
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
	colorEnabled bool
	header       lipgloss.Style
	brand        lipgloss.Style
	muted        lipgloss.Style
	section      lipgloss.Style
	project      lipgloss.Style
	selected     lipgloss.Style
	primary      lipgloss.Style
	command      lipgloss.Style
	success      lipgloss.Style
	attention    lipgloss.Style
	danger       lipgloss.Style
	stateMuted   lipgloss.Style
	stdout       lipgloss.Style
	stderr       lipgloss.Style
	key          lipgloss.Style
	border       lipgloss.Style
	strongBorder lipgloss.Style
	canvasColor  string
	primaryColor string
}

func newPresentation(colorEnabled bool) presentation {
	const (
		canvas    = "#0B1117"
		surface   = "#111B24"
		primary   = "#E7EDF4"
		muted     = "#91A0B2"
		accent    = "#75B8FF"
		success   = "#7CD992"
		attention = "#E6C66A"
		danger    = "#F08080"
		border    = "#334252"
		strong    = "#5E7892"
	)

	p := presentation{
		colorEnabled: colorEnabled,
		canvasColor:  canvas,
		primaryColor: primary,
		header:       lipgloss.NewStyle(),
		brand:        lipgloss.NewStyle(),
		muted:        lipgloss.NewStyle(),
		section:      lipgloss.NewStyle(),
		project:      lipgloss.NewStyle(),
		selected:     lipgloss.NewStyle(),
		primary:      lipgloss.NewStyle(),
		command:      lipgloss.NewStyle(),
		success:      lipgloss.NewStyle(),
		attention:    lipgloss.NewStyle(),
		danger:       lipgloss.NewStyle(),
		stateMuted:   lipgloss.NewStyle(),
		stdout:       lipgloss.NewStyle(),
		stderr:       lipgloss.NewStyle(),
		key:          lipgloss.NewStyle(),
		border:       lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()),
		strongBorder: lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()),
	}
	if !colorEnabled {
		return p
	}

	darkText := lipgloss.Color(canvas)
	p.header = p.header.Foreground(lipgloss.Color(primary)).Background(lipgloss.Color(surface))
	p.brand = p.brand.Bold(true).Foreground(darkText).Background(lipgloss.Color(accent))
	p.muted = p.muted.Foreground(lipgloss.Color(muted))
	p.section = p.section.Bold(true).Foreground(lipgloss.Color(accent))
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
	p.border = p.border.BorderForeground(lipgloss.Color(border)).Foreground(lipgloss.Color(primary)).Background(lipgloss.Color(surface)).ColorWhitespace(true)
	p.strongBorder = p.strongBorder.BorderForeground(lipgloss.Color(strong)).Foreground(lipgloss.Color(primary)).Background(lipgloss.Color(surface)).ColorWhitespace(true)
	return p
}

func colorEnabledFromEnvironment() bool {
	value, present := os.LookupEnv("NO_COLOR")
	return !present || value == ""
}

func (model Model) workspaceLayout() workspaceLayout {
	width, height := model.viewportSize()
	bodyHeight := max(1, height-headerHeight-shortcutBarHeight)
	if width >= wideWidth && height >= wideHeight {
		navigatorWidth := max(28, min(36, width/3))
		summaryHeight := 6
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
	summaryHeight := 6
	return workspaceLayout{
		bodyHeight:      bodyHeight,
		navigatorWidth:  width,
		navigatorHeight: navigatorHeight,
		summaryWidth:    width,
		summaryHeight:   summaryHeight,
		outputWidth:     width,
		outputHeight:    max(3, bodyHeight-navigatorHeight-summaryHeight),
	}
}

func (model Model) renderWorkspace() string {
	p := model.presentation
	width, _ := model.viewportSize()
	layout := model.workspaceLayout()
	header := p.renderHeader(width)
	navigator := p.renderNavigator(model, layout.navigatorWidth, layout.navigatorHeight)
	summary := p.renderSummary(model, layout.summaryWidth, layout.summaryHeight)
	output := p.renderWorkArea(model, layout.outputWidth, layout.outputHeight)
	workArea := lipgloss.JoinVertical(lipgloss.Left, summary, output)
	body := workArea
	if layout.wide {
		body = lipgloss.JoinHorizontal(lipgloss.Top, navigator, workArea)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, navigator, workArea)
	}
	footer := p.renderShortcutBar(model, width)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
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

func (p presentation) renderHeader(width int) string {
	brand := p.brand.Render(" DOVIK ")
	title := p.muted.Render(" LOCAL PROCESS SUPERVISOR")
	return p.header.Width(width).MaxWidth(width).Render(ansi.Truncate(brand+title, width, ""))
}

func (p presentation) renderNavigator(model Model, width, height int) string {
	capacity := max(1, height-3)
	position := ""
	if len(model.items) > 0 {
		position = fmt.Sprintf("  %d/%d", model.selected+1, len(model.items))
	}
	lines := []string{p.section.Render("PROCESSES") + p.muted.Render(position)}
	switch {
	case model.loading:
		lines = append(lines, p.muted.Render("Loading registered processes..."))
	case len(model.items) == 0:
		lines = append(lines, p.muted.Render("No process definitions."))
	default:
		start, end := model.navigatorRange(capacity)
		for index := start; index < end; index++ {
			item := model.items[index]
			label := fmt.Sprintf("  %s / %s", item.key.projectID, item.key.processID)
			if index == model.selected {
				label = fmt.Sprintf("> %s / %s", item.key.projectID, item.key.processID)
				lines = append(lines, p.selected.Width(max(1, width-2)).MaxWidth(max(1, width-2)).Render(label))
				continue
			}
			lines = append(lines, p.project.Render(label))
		}
	}
	return p.panel(width, height, false, lines)
}

func (p presentation) renderSummary(model Model, width, height int) string {
	lines := []string{p.section.Render("SELECTED PROCESS")}
	if len(model.items) == 0 {
		if model.loading {
			lines = append(lines, p.muted.Render("Waiting for registry data."))
		} else {
			lines = append(lines, "Use the CLI to register a project and process.")
		}
		return p.panel(width, height, true, lines)
	}

	item := model.items[model.selected]
	state := "loading"
	if model.statusKnown {
		state = string(model.currentState())
	}
	identity := p.primary.Bold(p.colorEnabled).Render(fmt.Sprintf("%s / %s", item.key.projectID, item.key.processID)) + "  " + p.renderState(state)
	lines = append(lines, identity)
	lines = append(lines, p.muted.Render("Command  ")+p.command.Render(item.command))
	if model.notice != "" {
		style := p.attention
		if model.diagnostic != "" {
			style = p.danger
		}
		lines = append(lines, style.Render(model.notice))
	} else {
		lines = append(lines, p.renderActions(model))
	}
	return p.panel(width, height, true, lines)
}

func (p presentation) renderActions(model Model) string {
	prefix := p.muted.Render("Action   ")
	if model.pending {
		return prefix + p.attention.Render("PENDING")
	}
	if !model.statusKnown {
		return prefix + p.muted.Render("Unavailable")
	}
	if model.currentState().IsActive() {
		return prefix + p.renderKey("x", "Stop") + "  " + p.renderKey("r", "Restart")
	}
	return prefix + p.renderKey("s", "Start")
}

func (p presentation) renderKey(key, action string) string {
	return p.key.Render("["+key+"]") + " " + action
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
		return p.panel(width, height, false, p.helpLines())
	}
	if model.showDetail {
		return p.panel(width, height, false, p.detailLines(model))
	}
	return p.panel(width, height, false, p.outputLines(model))
}

func (p presentation) outputLines(model Model) []string {
	lines := []string{p.section.Render("OUTPUT")}
	if model.truncated {
		lines = append(lines, p.attention.Render("[Older output is unavailable]"))
	}
	visible := model.visibleOutputLines()
	if len(visible) == 0 {
		lines = append(lines, p.muted.Render("No output yet."))
		return lines
	}
	for _, line := range visible {
		prefix := p.stdout.Render("[stdout]")
		if line.stream == supervision.OutputStreamStderr {
			prefix = p.stderr.Render("[stderr]")
		}
		lines = append(lines, prefix+" "+line.text)
	}
	return lines
}

func (p presentation) detailLines(model Model) []string {
	lines := []string{p.section.Render("RUNTIME DETAILS")}
	if !model.hasRuntime {
		return append(lines, p.muted.Render("No runtime has started."))
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
		fmt.Sprintf("Instance  %s    PID  %s", model.runtime.InstanceID, pid),
		fmt.Sprintf("Started   %s", formatTime(model.runtime.StartedAt)),
		fmt.Sprintf("Ended     %s    Exit  %s    Reason  %s", formatTime(model.runtime.FinishedAt), exitCode, emptyAsDash(model.runtime.TerminationReason)),
	)
	if model.diagnostic != "" {
		lines = append(lines, p.danger.Render("Diagnostic  ")+model.diagnostic)
	}
	return lines
}

func (p presentation) helpLines() []string {
	return []string{
		p.section.Render("KEYBOARD HELP"),
		p.key.Render("up/down, j/k") + "  Select process",
		p.key.Render("s / x / r") + "    Start / stop / restart",
		p.key.Render("PgUp/PgDn") + "    Scroll output",
		p.key.Render("l / d / ? / q") + " Refresh / details / close help / quit",
	}
}

func (p presentation) renderShortcutBar(model Model, width int) string {
	shortcuts := p.renderKey("j/k", "Select")
	if len(model.items) > 0 && model.statusKnown {
		if model.currentState().IsActive() {
			shortcuts += "  " + p.renderKey("x", "Stop") + "  " + p.renderKey("r", "Restart")
		} else {
			shortcuts += "  " + p.renderKey("s", "Start")
		}
	}
	detailsLabel := "Details"
	if model.showDetail && !model.showHelp {
		detailsLabel = "Close details"
	}
	helpLabel := "Help"
	if model.showHelp {
		helpLabel = "Close help"
	}
	shortcuts += "  " + p.renderKey("d", detailsLabel) + "  " + p.renderKey("?", helpLabel) + "  " + p.renderKey("q", "Quit")
	return p.header.Width(width).MaxWidth(width).Render(ansi.Truncate(" "+shortcuts, width, ""))
}

func (p presentation) panel(width, height int, strong bool, lines []string) string {
	innerWidth := max(1, width-2)
	innerHeight := max(1, height-2)
	fitted := make([]string, 0, min(len(lines), innerHeight))
	for _, line := range lines {
		if len(fitted) == innerHeight {
			break
		}
		fitted = append(fitted, ansi.Truncate(line, innerWidth, ""))
	}
	style := p.border
	if strong {
		style = p.strongBorder
	}
	return style.Width(width).Height(height).Render(strings.Join(fitted, "\n"))
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
