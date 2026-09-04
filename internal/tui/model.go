package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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
	ctx         context.Context
	client      operatorclient.Client
	items       []processItem
	selected    int
	generation  uint64
	width       int
	height      int
	loading     bool
	refreshing  bool
	pending     bool
	runtime     supervision.ProcessRuntime
	hasRuntime  bool
	statusKnown bool
	events      []supervision.OutputEvent
	truncated   bool
	lastSeq     uint64
	scroll      int
	showHelp    bool
	showDetail  bool
	notice      string
	diagnostic  string
}

// NewModel creates a model backed exclusively by the shared daemon client.
func NewModel(ctx context.Context, client operatorclient.Client) Model {
	return Model{ctx: ctx, client: client, loading: true}
}

func (model Model) Init() tea.Cmd {
	return tea.Batch(model.loadRegistryCmd(), tickCmd())
}

func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height
		return model, nil
	case tea.KeyPressMsg:
		return model.updateKey(message.String())
	case registryLoadedMsg:
		model.loading = false
		if message.err != nil {
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
		if len(model.items) == 0 {
			return model, nil
		}
		model.refreshing = true
		return model, model.refreshCmd()
	case refreshMsg:
		if message.key != model.selectedKey() || message.generation != model.generation {
			return model, nil
		}
		model.refreshing = false
		if message.err != nil {
			model.notice = "Process state is unavailable. Press l to retry."
			model.diagnostic = message.err.Error()
			return model, nil
		}
		model.applyRuntime(message.runtime, message.exists)
		model.statusKnown = true
		model.appendEvents(message.tail.Events)
		model.truncated = model.truncated || message.tail.Truncated
		model.notice = ""
		model.diagnostic = ""
		return model, nil
	case actionMsg:
		model.pending = false
		if message.key != model.selectedKey() || message.generation != model.generation {
			return model, nil
		}
		if message.err != nil {
			model.notice = strings.ToUpper(message.action[:1]) + message.action[1:] + " did not complete. Use the same key to retry."
			model.diagnostic = message.err.Error()
			return model, nil
		}
		model.applyRuntime(message.runtime, true)
		model.statusKnown = true
		model.notice = ""
		model.diagnostic = ""
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
		model.scroll = min(len(model.events), model.scroll+model.pageHeight())
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
	model.notice = action + " pending..."
	return model, model.actionCmd(action)
}

func (model Model) View() tea.View {
	content := model.render()
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (model Model) render() string {
	if model.width > 0 && model.height > 0 && (model.width < minimumWidth || model.height < minimumHeight) {
		return fmt.Sprintf("Dovik\n\nTerminal too small: need at least %dx%d, current %dx%d.\n\n? help  q quit", minimumWidth, minimumHeight, model.width, model.height)
	}

	var builder strings.Builder
	builder.WriteString("Dovik - local process supervisor\n\n")
	if model.loading {
		builder.WriteString("Loading registered processes...\n")
	} else if len(model.items) == 0 {
		builder.WriteString("No process definitions are registered.\nUse CLI project and process commands to configure Dovik.\n")
	} else {
		builder.WriteString("Processes\n")
		start, end := model.navigatorRange()
		if start > 0 {
			builder.WriteString("  ...\n")
		}
		var renderedProject supervision.ProjectID
		for index := start; index < end; index++ {
			item := model.items[index]
			if item.key.projectID != renderedProject {
				fmt.Fprintf(&builder, "  %s\n", item.key.projectID)
				renderedProject = item.key.projectID
			}
			marker := "    "
			if index == model.selected {
				marker = ">   "
			}
			fmt.Fprintf(&builder, "%s%s  %s\n", marker, item.key.processID, item.command)
		}
		if end < len(model.items) {
			builder.WriteString("  ...\n")
		}
		builder.WriteString("\nSelected\n")
		item := model.items[model.selected]
		stateLabel := "loading"
		if model.statusKnown {
			stateLabel = string(model.currentState())
		}
		fmt.Fprintf(&builder, "%s/%s  state=%s", item.key.projectID, item.key.processID, stateLabel)
		if model.pending {
			builder.WriteString("  action=pending")
		}
		builder.WriteString("\n")
		if model.statusKnown && model.currentState().IsActive() {
			builder.WriteString("Actions: x stop  r restart\n")
		} else if model.statusKnown {
			builder.WriteString("Actions: s start\n")
		} else {
			builder.WriteString("Actions: waiting for process state\n")
		}
		if model.showDetail {
			builder.WriteString(model.renderDetails())
		}
		builder.WriteString("\nOutput\n")
		if model.truncated {
			builder.WriteString("[output truncated: older events are unavailable]\n")
		}
		for _, event := range model.visibleEvents() {
			fmt.Fprintf(&builder, "[%s] %s", event.Stream, event.Data)
			if len(event.Data) == 0 || event.Data[len(event.Data)-1] != '\n' {
				builder.WriteByte('\n')
			}
		}
	}
	if model.notice != "" {
		fmt.Fprintf(&builder, "\n%s\n", model.notice)
	}
	if model.showDetail && model.diagnostic != "" {
		fmt.Fprintf(&builder, "Diagnostic: %s\n", model.diagnostic)
	}
	if model.showHelp {
		builder.WriteString("\nKeys\nup/k down/j select  s start  x stop  r restart  l refresh\nd details  pgup/pgdown output  ? close help  q quit\n")
	} else {
		builder.WriteString("\nup/down select  s start  x stop  r restart  l refresh  d details  ? help  q quit\n")
	}
	return builder.String()
}

func (model Model) renderDetails() string {
	if !model.hasRuntime {
		return "Details: no runtime has started\n"
	}
	pid := "-"
	if model.runtime.PID != nil {
		pid = fmt.Sprint(*model.runtime.PID)
	}
	started := formatTime(model.runtime.StartedAt)
	finished := formatTime(model.runtime.FinishedAt)
	exitCode := "-"
	if model.runtime.ExitCode != nil {
		exitCode = fmt.Sprint(*model.runtime.ExitCode)
	}
	return fmt.Sprintf("Details: instance=%s pid=%s started=%s finished=%s exit=%s reason=%s\n",
		model.runtime.InstanceID, pid, started, finished, exitCode, emptyAsDash(model.runtime.TerminationReason))
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

func (model Model) visibleEvents() []supervision.OutputEvent {
	height := model.pageHeight()
	end := max(0, len(model.events)-model.scroll)
	start := max(0, end-height)
	return model.events[start:end]
}

func (model Model) pageHeight() int {
	if model.height <= 0 {
		return 8
	}
	return max(3, model.height-14-model.navigatorHeight())
}

func (model Model) navigatorHeight() int {
	if model.height <= 0 {
		return min(8, len(model.items))
	}
	return max(3, min(len(model.items), model.height/3))
}

func (model Model) navigatorRange() (int, int) {
	height := model.navigatorHeight()
	start := max(0, model.selected-height/2)
	end := min(len(model.items), start+height)
	start = max(0, end-height)
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
					command: process.Command,
				})
			}
		}
		return registryLoadedMsg{items: items}
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
