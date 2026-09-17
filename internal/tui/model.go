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
	"github.com/MrMaxie/dovik/internal/identity"
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
	state   supervision.ProcessState
	known   bool
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

type daemonLaunchMsg struct {
	items []processItem
	err   error
}

// DaemonLauncher starts a daemon without owning its lifetime.
type DaemonLauncher interface {
	Start() (int, error)
}

type registryOutcome uint8

const (
	registryLoading registryOutcome = iota
	registryEmpty
	registryPopulated
	registryUnavailable
)

type workspaceTab uint8

const (
	workspaceProcesses workspaceTab = iota
	workspacePersonas
	workspaceProjects
	workspaceTabCount
)

// Model is the terminal-independent TUI state machine.
type Model struct {
	identities         identity.Snapshot
	identityError      string
	identityLoading    bool
	identitySelected   int
	personaSelected    int
	activeWorkspace    workspaceTab
	editor             *editorState
	confirmation       *confirmationState
	savingIdentity     bool
	ctx                context.Context
	client             operatorclient.Client
	daemonLauncher     DaemonLauncher
	daemonStarting     bool
	daemonReadyTimeout time.Duration
	daemonRetryDelay   time.Duration
	items              []processItem
	selected           int
	generation         uint64
	width              int
	height             int
	loading            bool
	registry           registryOutcome
	refreshing         bool
	pending            bool
	runtime            supervision.ProcessRuntime
	hasRuntime         bool
	statusKnown        bool
	events             []supervision.OutputEvent
	truncated          bool
	lastSeq            uint64
	scroll             int
	showHelp           bool
	showDetail         bool
	showOutput         bool
	followOutput       bool
	notice             string
	diagnostic         string
	presentation       presentation
}

// NewModel creates a model backed exclusively by the shared daemon client.
func NewModel(ctx context.Context, client operatorclient.Client) Model {
	return NewModelWithDaemonLauncher(ctx, client, nil)
}

// NewModelWithDaemonLauncher creates a model that can explicitly start an
// unavailable local daemon.
func NewModelWithDaemonLauncher(ctx context.Context, client operatorclient.Client, launcher DaemonLauncher) Model {
	return Model{
		ctx:                ctx,
		client:             client,
		daemonLauncher:     launcher,
		activeWorkspace:    workspaceProcesses,
		identityLoading:    true,
		daemonReadyTimeout: requestTimeout,
		daemonRetryDelay:   100 * time.Millisecond,
		loading:            true,
		presentation:       newPresentation(colorEnabledFromEnvironment()),
	}
}

func (model Model) Init() tea.Cmd {
	devLog("model.initialized")
	return tea.Batch(model.loadRegistryCmd(), model.loadIdentityCmd(), tickCmd())
}

func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case identityLoadedMsg:
		model.identityLoading = false
		model.identities = message.snapshot
		model.identityError = ""
		if message.err != nil {
			model.identityError = message.err.Error()
		}
		model.identitySelected = min(model.identitySelected, max(0, len(model.identities.State.Projects)-1))
		model.personaSelected = min(model.personaSelected, max(0, len(model.identities.State.Personas)-1))
		return model, nil
	case identitySavedMsg:
		model.identityLoading = false
		model.savingIdentity = false
		if message.err != nil {
			model.notice = message.err.Error()
			return model, nil
		}
		model.identities = message.snapshot
		model.identityError = ""
		model.notice = "saved"
		model.editor = nil
		return model, nil
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height
		devLog("terminal.size", "cols", message.Width, "rows", message.Height)
		if model.editor != nil {
			return model.updateEditor(message)
		}
		return model, nil
	case tea.KeyPressMsg:
		devLogKey(message.String())
		if model.confirmation != nil {
			return model.updateConfirmation(message.String())
		}
		if model.editor != nil {
			return model.updateEditor(message)
		}
		return model.updateKey(message.String())
	case registryLoadedMsg:
		return model.applyRegistryLoaded(message)
	case daemonLaunchMsg:
		model.daemonStarting = false
		if message.err != nil {
			devLog("daemon.launch.failed", "error", message.err.Error())
			model.registry = registryUnavailable
			model.statusKnown = false
			model.notice = "Daemon did not start or become available. Press s to retry."
			model.diagnostic = message.err.Error()
			return model, nil
		}
		devLog("daemon.launch.ready")
		return model.applyRegistryLoaded(registryLoadedMsg{items: message.items})
	case refreshMsg:
		if message.key != model.selectedKey() || message.generation != model.generation {
			devLog("refresh.ignored", "reason", "stale selection")
			return model, nil
		}
		model.refreshing = false
		if message.err != nil {
			devLog("refresh.failed", "error", message.err.Error())
			model.statusKnown = false
			model.notice = "Process state is unavailable. Press l to retry."
			model.diagnostic = message.err.Error()
			return model, nil
		}
		model.applyRuntime(message.runtime, message.exists)
		model.statusKnown = true
		model.appendEvents(message.tail.Events)
		model.truncated = model.truncated || message.tail.Truncated
		if model.registry != registryUnavailable && model.notice == "Process state is unavailable. Press l to retry." {
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
		allowRefresh := !model.showOutput || model.followOutput
		if allowRefresh && len(model.items) > 0 && !model.loading && !model.daemonStarting && model.registry != registryUnavailable && !model.refreshing && !model.pending {
			model.refreshing = true
			commands = append(commands, model.refreshCmd())
		}
		return model, tea.Batch(commands...)
	}
	if model.editor != nil {
		return model.updateEditor(message)
	}
	return model, nil
}

func (model Model) updateKey(key string) (tea.Model, tea.Cmd) {
	if model.showOutput {
		switch key {
		case "esc", "q", "o":
			model.showOutput = false
			model.followOutput = false
			model.scroll = 0
			return model, nil
		case "f":
			model.followOutput = !model.followOutput
			if model.followOutput {
				model.scroll = 0
				if !model.refreshing {
					model.refreshing = true
					return model, model.refreshCmd()
				}
			}
			return model, nil
		case "up":
			model.followOutput = false
			model.scroll = min(model.fullOutputMaxScroll(), model.scroll+1)
			return model, nil
		case "down":
			model.scroll = max(0, model.scroll-1)
			return model, nil
		case "pgup":
			model.followOutput = false
			model.scroll = min(model.fullOutputMaxScroll(), model.scroll+model.fullOutputPageHeight())
			return model, nil
		case "pgdown":
			model.scroll = max(0, model.scroll-model.fullOutputPageHeight())
			return model, nil
		case "home":
			model.followOutput = false
			model.scroll = model.fullOutputMaxScroll()
			return model, nil
		case "end":
			model.scroll = 0
			return model, nil
		case "ctrl+c":
			return model, tea.Quit
		default:
			return model, nil
		}
	}
	switch key {
	case "left":
		return model.switchWorkspace(-1)
	case "right":
		return model.switchWorkspace(1)
	case "q", "ctrl+c":
		return model, tea.Quit
	}

	if model.activeWorkspace == workspaceProjects {
		switch key {
		case "esc":
			model.activeWorkspace = workspaceProcesses
			return model, nil
		case "down":
			model.identitySelected = min(model.identitySelected+1, max(0, len(model.identities.State.Projects)-1))
			return model, nil
		case "up":
			model.identitySelected = max(0, model.identitySelected-1)
			return model, nil
		case "enter":
			if model.identityLoading || model.identityError != "" {
				return model, nil
			}
			if model.pending || model.daemonStarting || model.registry == registryUnavailable {
				return model, nil
			}
			return model.openProjectEditor()
		case "l":
			model.identityLoading = true
			return model, model.loadIdentityCmd()
		default:
			return model, nil
		}
	}

	if model.activeWorkspace == workspacePersonas {
		switch key {
		case "esc":
			model.activeWorkspace = workspaceProcesses
			return model, nil
		case "down":
			model.personaSelected = min(model.personaSelected+1, max(0, len(model.identities.State.Personas)-1))
			return model, nil
		case "up":
			model.personaSelected = max(0, model.personaSelected-1)
			return model, nil
		case "enter":
			if model.identityLoading || model.identityError != "" || len(model.identities.State.Personas) == 0 {
				return model, nil
			}
			return model.openPersonaEditor()
		case "l":
			model.identityLoading = true
			return model, model.loadIdentityCmd()
		default:
			return model, nil
		}
	}

	switch key {
	case "i":
		model.identityLoading = true
		model.activeWorkspace = workspaceProjects
		return model, model.loadIdentityCmd()
	case "?":
		model.showHelp = !model.showHelp
		return model, nil
	case "d":
		if len(model.items) > 0 || model.diagnostic != "" {
			model.showDetail = !model.showDetail
		}
		return model, nil
	case "o":
		if len(model.items) > 0 && model.registry != registryUnavailable {
			model.showOutput = true
			model.followOutput = false
			model.scroll = 0
		}
		return model, nil
	case "enter":
		if !model.pending && !model.loading && model.registry != registryUnavailable && len(model.items) > 0 {
			return model.openProcessEditor()
		}
	case "up":
		if !model.pending && !model.loading && model.registry != registryUnavailable && model.selected > 0 {
			model.selected--
			model.resetSelection()
			model.refreshing = true
			return model, model.refreshCmd()
		}
	case "down":
		if !model.pending && !model.loading && model.registry != registryUnavailable && model.selected+1 < len(model.items) {
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
		if !model.pending && !model.loading && !model.daemonStarting {
			model.loading = true
			model.registry = registryLoading
			model.generation++
			model.refreshing = false
			return model, model.loadRegistryCmd()
		}
	case "s":
		if model.registry == registryUnavailable {
			return model.beginDaemonStart()
		}
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
	case "q", "ctrl+c", "esc", "enter", "?", "d", "i", "o", "f", "left", "right", "up", "down", "pgup", "pgdown", "home", "end", "l", "s", "x", "r":
		devLog("input.key", "key", key)
	}
}

func (model Model) switchWorkspace(direction int) (tea.Model, tea.Cmd) {
	order := []workspaceTab{workspacePersonas, workspaceProjects, workspaceProcesses}
	current := 0
	for index, workspace := range order {
		if workspace == model.activeWorkspace {
			current = index
			break
		}
	}
	next := (current + direction + len(order)) % len(order)
	model.activeWorkspace = order[next]
	if model.activeWorkspace == workspaceProcesses {
		return model, nil
	}
	model.identityLoading = true
	return model, model.loadIdentityCmd()
}

func (model Model) beginAction(action string) (tea.Model, tea.Cmd) {
	if model.pending || model.loading || model.daemonStarting || model.registry == registryUnavailable || len(model.items) == 0 {
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
	if model.confirmation != nil || model.editor != nil {
		return model.renderEditorWorkspace()
	}
	switch model.activeWorkspace {
	case workspaceProjects:
		return model.identityView()
	case workspacePersonas:
		return model.personaView()
	default:
		return model.renderProcessWorkspace()
	}
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
	if model.selected >= 0 && model.selected < len(model.items) {
		model.items[model.selected].known = true
		model.items[model.selected].state = supervision.ProcessStateStopped
		if exists {
			model.items[model.selected].state = runtimeState.State
		}
	}
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
			if part == "" {
				continue
			}
			lines = append(lines, outputLine{stream: event.Stream, text: part})
		}
	}
	return lines
}

func (model Model) visibleOutputLines(limit int) []outputLine {
	lines := model.outputLines()
	height := max(1, limit)
	end := max(0, len(lines)-model.scroll)
	start := max(0, end-height)
	return lines[start:end]
}

func (model Model) pageHeight() int {
	if model.height <= 0 || model.width <= 0 {
		return 8
	}
	layout := model.processWorkspaceLayout()
	reserved := 3
	if model.truncated {
		reserved++
	}
	return max(1, layout.outputHeight-reserved)
}

func (model Model) fullOutputPageHeight() int {
	reserved := 6
	if model.truncated {
		reserved++
	}
	return max(1, model.processWorkspaceLayout().bodyHeight-reserved)
}

func (model Model) fullOutputMaxScroll() int {
	return max(0, len(model.outputLines())-model.fullOutputPageHeight())
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
		items, err := model.loadRegistry(ctx)
		return registryLoadedMsg{items: items, err: err}
	}
}

func (model Model) loadRegistry(ctx context.Context) ([]processItem, error) {
	projects, err := model.client.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]processItem, 0)
	for _, project := range projects {
		processes, err := model.client.ListProcesses(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		for _, process := range processes {
			runtimeState, exists, statusErr := model.client.Status(ctx, process.ProjectID, process.ID)
			if statusErr != nil {
				return nil, statusErr
			}
			state := supervision.ProcessStateStopped
			if exists {
				state = runtimeState.State
			}
			items = append(items, processItem{
				key:     processKey{projectID: process.ProjectID, processID: process.ID},
				command: formatCommand(process.Command, process.Arguments),
				state:   state,
				known:   true,
			})
		}
	}
	return items, nil
}

func (model Model) applyRegistryLoaded(message registryLoadedMsg) (tea.Model, tea.Cmd) {
	model.loading = false
	if message.err != nil {
		model.registry = registryUnavailable
		devLog("registry.failed", "error", message.err.Error())
		model.statusKnown = false
		model.notice = ""
		if model.daemonLauncher == nil {
			model.notice = "Process state is unavailable. Press l to retry."
		}
		model.diagnostic = message.err.Error()
		return model, nil
	}
	previous := model.selectedKey()
	model.items = message.items
	model.registry = registryPopulated
	if len(model.items) == 0 {
		model.registry = registryEmpty
		model.showDetail = false
	}
	model.selected = indexOfKey(model.items, previous)
	if model.selected < 0 && len(model.items) > 0 {
		model.selected = 0
	}
	if model.selectedKey() != previous || len(model.items) == 0 {
		model.resetSelection()
	}
	model.notice = ""
	model.diagnostic = ""
	devLog("registry.loaded", "processes", len(model.items))
	if len(model.items) == 0 {
		return model, nil
	}
	model.refreshing = true
	return model, model.refreshCmd()
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
