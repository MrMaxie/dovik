package tui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/MrMaxie/dovik/internal/supervision"
)

const (
	minimumWidth   = 64
	minimumHeight  = 18
	refreshEvery   = time.Second
	requestTimeout = 5 * time.Second
	outputLimit    = 400
)

type processKey struct {
	projectID supervision.ProjectID
	processID supervision.ProcessID
}

type processItem struct {
	key     processKey
	command string
}

type outputLine struct {
	stream supervision.OutputStream
	text   string
}

type registryLoadedMsg struct {
	items []processItem
	err   error
}

type refreshMsg struct {
	key        processKey
	generation uint64
	runtime    supervision.ProcessRuntime
	exists     bool
	tail       supervision.OutputTail
	err        error
}

type actionMsg struct {
	key        processKey
	generation uint64
	action     string
	runtime    supervision.ProcessRuntime
	err        error
}

type tickMsg time.Time

// Model is the terminal-independent TUI state machine.
type Model struct {
	ctx          context.Context
	client       operatorclient.Client
	items        []processItem
	selected     int
	generation   uint64
	width        int
	height       int
	loading      bool
	refreshing   bool
	pending      bool
	runtime      supervision.ProcessRuntime
	hasRuntime   bool
	statusKnown  bool
	events       []supervision.OutputEvent
	truncated    bool
	lastSeq      uint64
	scroll       int
	showHelp     bool
	showDetail   bool
	notice       string
	diagnostic   string
	presentation presentation
}

// NewModel creates a model backed exclusively by the shared daemon client.
func NewModel(ctx context.Context, client operatorclient.Client) Model {
	return Model{ctx: ctx, client: client, loading: true, presentation: newPresentation(colorEnabledFromEnvironment())}
}

func (model Model) Init() tea.Cmd {
	devLog("model.initialized")
	return tea.Batch(model.loadRegistryCmd(), tickCmd())
}

func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height
		devLog("terminal.size", "cols", message.Width, "rows", message.Height)
		return model, nil
	case tea.KeyPressMsg:
		devLogKey(message.String())
		return model.updateKey(message.String())
	case registryLoadedMsg:
		model.loading = false
		if message.err != nil {
			devLog("registry.failed", "error", message.err.Error())
			model.statusKnown = false
			model.notice = "Process state is unavailable. Press l to retry."
			model.diagnostic = message.err.Error()
			return model, nil
		}
		previous := model.selectedKey()
		model.items = message.items
		model.selected = indexOfKey(model.items, previous)
		if model.selected < 0 && len(model.items) > 0 {
			model.selected = 0
		}
		model.resetSelection()
		model.notice = ""
		model.diagnostic = ""
		devLog("registry.loaded", "processes", len(model.items))
		if len(model.items) == 0 {
			return model, nil
		}
		model.refreshing = true
		return model, model.refreshCmd()
	case refreshMsg:
		if message.key != model.selectedKey() || message.generation != model.generation {
			devLog("refresh.ignored", "reason", "stale selection")
			return model, nil
		}
		model.refreshing = false
		if message.err != nil {
			devLog("refresh.failed", "error", message.err.Error())
			model.notice = "Process state is unavailable. Press l to retry."
			model.diagnostic = message.err.Error()
			return model, nil
		}
		model.applyRuntime(message.runtime, message.exists)
		model.statusKnown = true
		model.appendEvents(message.tail.Events)
		model.truncated = model.truncated || message.tail.Truncated
		if model.notice == "Process state is unavailable. Press l to retry." {
			model.notice = ""
			model.diagnostic = ""
		}
		devLog("refresh.completed", "project", message.key.projectID, "process", message.key.processID, "state", model.currentState(), "events", len(message.tail.Events), "truncated", message.tail.Truncated)
		return model, nil
	case actionMsg:
		model.pending = false
		if message.key != model.selectedKey() || message.generation != model.generation {
			return model, nil
		}
		if message.err != nil {
			devLog("action.failed", "action", message.action, "error", message.err.Error())
			model.notice = fmt.Sprintf("%s did not complete. Press %s to retry.", actionLabel(message.action), actionKey(message.action))
			model.diagnostic = message.err.Error()
			return model, nil
		}
		model.applyRuntime(message.runtime, true)
		model.statusKnown = true
		model.notice = ""
		model.diagnostic = ""
		devLog("action.completed", "action", message.action, "project", message.key.projectID, "process", message.key.processID, "state", model.currentState())
		if !model.refreshing {
			model.refreshing = true
			return model, model.refreshCmd()
		}
		return model, nil
	case tickMsg:
		commands := []tea.Cmd{tickCmd()}
		if len(model.items) > 0 && !model.refreshing && !model.pending {
			model.refreshing = true
			commands = append(commands, model.refreshCmd())
		}
		return model, tea.Batch(commands...)
	}
	return model, nil
}

func (model Model) updateKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		return model, tea.Quit
	case "?":
		model.showHelp = !model.showHelp
		return model, nil
	case "d":
		model.showDetail = !model.showDetail
		return model, nil
	case "up", "k":
		if !model.pending && model.selected > 0 {
			model.selected--
			model.resetSelection()
			model.refreshing = true
			return model, model.refreshCmd()
		}
	case "down", "j":
		if !model.pending && model.selected+1 < len(model.items) {
			model.selected++
			model.resetSelection()
			model.refreshing = true
			return model, model.refreshCmd()
		}
	case "pgup":
		model.scroll = min(len(model.outputLines()), model.scroll+model.pageHeight())
		return model, nil
	case "pgdown":
		model.scroll = max(0, model.scroll-model.pageHeight())
		return model, nil
	case "l":
		if !model.pending {
			model.loading = true
			return model, model.loadRegistryCmd()
		}
	case "s":
		return model.beginAction("start")
	case "x":
		return model.beginAction("stop")
	case "r":
		return model.beginAction("restart")
	}
	return model, nil
}

func devLogKey(key string) {
	switch key {
	case "q", "ctrl+c", "?", "d", "up", "k", "down", "j", "pgup", "pgdown", "l", "s", "x", "r":
		devLog("input.key", "key", key)
	}
}

func (model Model) beginAction(action string) (tea.Model, tea.Cmd) {
	if model.pending || len(model.items) == 0 {
		return model, nil
	}
	if !model.statusKnown {
		model.notice = "Process state is unavailable. Press l to retry."
		return model, nil
	}
	if !model.actionAvailable(action) {
		model.notice = fmt.Sprintf("%s is not available while state is %s", action, model.currentState())
		return model, nil
	}
	model.generation++
	model.refreshing = false
	model.pending = true
	model.notice = actionLabel(action) + " pending..."
	model.diagnostic = ""
	devLog("action.started", "action", action, "project", model.items[model.selected].key.projectID, "process", model.items[model.selected].key.processID)
	return model, model.actionCmd(action)
}

func (model Model) View() tea.View {
	content := model.render()
	view := tea.NewView(content)
	view.AltScreen = true
	if model.presentation.colorEnabled {
		view.BackgroundColor = lipgloss.Color(model.presentation.canvasColor)
		view.ForegroundColor = lipgloss.Color(model.presentation.primaryColor)
	}
	return view
}

func (model Model) render() string {
	if model.width > 0 && model.height > 0 && (model.width < minimumWidth || model.height < minimumHeight) {
		return model.renderSmallTerminal()
	}
	return model.renderWorkspace()
}

func (model *Model) resetSelection() {
	model.generation++
	model.runtime = supervision.ProcessRuntime{}
	model.hasRuntime = false
	model.statusKnown = false
	model.events = nil
	model.truncated = false
	model.lastSeq = 0
	model.scroll = 0
	model.notice = ""
	model.diagnostic = ""
}

func (model *Model) applyRuntime(runtimeState supervision.ProcessRuntime, exists bool) {
	if exists && model.hasRuntime && runtimeState.InstanceID != model.runtime.InstanceID {
		model.events = nil
		model.truncated = false
		model.lastSeq = 0
		model.scroll = 0
	}
	model.runtime = runtimeState
	model.hasRuntime = exists
}

func (model *Model) appendEvents(events []supervision.OutputEvent) {
	sort.SliceStable(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })
	for _, event := range events {
		if event.Sequence <= model.lastSeq {
			continue
		}
		event.Data = append([]byte(nil), event.Data...)
		model.events = append(model.events, event)
		model.lastSeq = event.Sequence
	}
	if len(model.events) > outputLimit {
		model.events = append([]supervision.OutputEvent(nil), model.events[len(model.events)-outputLimit:]...)
	}
}

func (model Model) outputLines() []outputLine {
	lines := make([]outputLine, 0, len(model.events))
	for _, event := range model.events {
		data := strings.ReplaceAll(string(event.Data), "\r\n", "\n")
		data = strings.TrimSuffix(data, "\n")
		parts := strings.Split(data, "\n")
		for _, part := range parts {
			lines = append(lines, outputLine{stream: event.Stream, text: part})
		}
	}
	return lines
}

func (model Model) visibleOutputLines() []outputLine {
	lines := model.outputLines()
	height := model.pageHeight()
	end := max(0, len(lines)-model.scroll)
	start := max(0, end-height)
	return lines[start:end]
}

func (model Model) pageHeight() int {
	if model.height <= 0 || model.width <= 0 {
		return 8
	}
	layout := model.workspaceLayout()
	reserved := 3
	if model.truncated {
		reserved++
	}
	return max(1, layout.outputHeight-reserved)
}

func (model Model) navigatorRange(capacity int) (int, int) {
	start := max(0, model.selected-capacity/2)
	end := min(len(model.items), start+capacity)
	start = max(0, end-capacity)
	return start, end
}

func (model Model) selectedKey() processKey {
	if model.selected < 0 || model.selected >= len(model.items) {
		return processKey{}
	}
	return model.items[model.selected].key
}

func (model Model) currentState() supervision.ProcessState {
	if !model.hasRuntime {
		return supervision.ProcessStateStopped
	}
	return model.runtime.State
}

func (model Model) actionAvailable(action string) bool {
	if !model.statusKnown {
		return false
	}
	active := model.currentState().IsActive()
	switch action {
	case "start":
		return !active
	case "stop", "restart":
		return active
	default:
		return false
	}
}

func (model Model) loadRegistryCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(model.ctx, requestTimeout)
		defer cancel()
		projects, err := model.client.ListProjects(ctx)
		if err != nil {
			return registryLoadedMsg{err: err}
		}
		items := make([]processItem, 0)
		for _, project := range projects {
			processes, err := model.client.ListProcesses(ctx, project.ID)
			if err != nil {
				return registryLoadedMsg{err: err}
			}
			for _, process := range processes {
				items = append(items, processItem{
					key:     processKey{projectID: process.ProjectID, processID: process.ID},
					command: formatCommand(process.Command, process.Arguments),
				})
			}
		}
		return registryLoadedMsg{items: items}
	}
}

func formatCommand(command string, arguments []string) string {
	parts := make([]string, 0, len(arguments)+1)
	parts = append(parts, command)
	for _, argument := range arguments {
		if strings.ContainsAny(argument, " \t\"") {
			parts = append(parts, strconv.Quote(argument))
		} else {
			parts = append(parts, argument)
		}
	}
	return strings.Join(parts, " ")
}

func actionLabel(action string) string {
	if action == "" {
		return "Action"
	}
	return strings.ToUpper(action[:1]) + action[1:]
}

func actionKey(action string) string {
	switch action {
	case "start":
		return "s"
	case "stop":
		return "x"
	case "restart":
		return "r"
	default:
		return "the action key"
	}
}

func (model Model) refreshCmd() tea.Cmd {
	key := model.selectedKey()
	generation := model.generation
	after := model.lastSeq
	previousInstance := model.runtime.InstanceID
	hadRuntime := model.hasRuntime
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(model.ctx, requestTimeout)
		defer cancel()
		runtimeState, exists, err := model.client.Status(ctx, key.projectID, key.processID)
		if err != nil {
			return refreshMsg{key: key, generation: generation, err: err}
		}
		if exists && (!hadRuntime || runtimeState.InstanceID != previousInstance) {
			after = 0
		}
		tail, err := model.client.Logs(ctx, key.projectID, key.processID, after, 200)
		return refreshMsg{key: key, generation: generation, runtime: runtimeState, exists: exists, tail: tail, err: err}
	}
}

func (model Model) actionCmd(action string) tea.Cmd {
	key := model.selectedKey()
	generation := model.generation
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(model.ctx, requestTimeout)
		defer cancel()
		var runtimeState supervision.ProcessRuntime
		var err error
		switch action {
		case "start":
			runtimeState, err = model.client.Start(ctx, key.projectID, key.processID)
		case "stop":
			runtimeState, err = model.client.Stop(ctx, key.projectID, key.processID)
		case "restart":
			runtimeState, err = model.client.Restart(ctx, key.projectID, key.processID)
		}
		return actionMsg{key: key, generation: generation, action: action, runtime: runtimeState, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshEvery, func(now time.Time) tea.Msg { return tickMsg(now) })
}

func indexOfKey(items []processItem, key processKey) int {
	for index, item := range items {
		if item.key == key {
			return index
		}
	}
	return -1
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.Format(time.RFC3339)
}

func emptyAsDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
