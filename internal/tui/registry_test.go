package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/supervision"
)

func TestRegistryFailureIsNotEmptyAndRetryRecovers(t *testing.T) {
	client := &fakeClient{listErr: errors.New("private transport detail")}
	model := NewModel(context.Background(), client)
	model.presentation = newPresentation(false)
	if view := model.View().Content; !strings.Contains(view, "Loading processes") || strings.Contains(view, "No processes registered") {
		t.Fatalf("initial state: %s", view)
	}
	failed, _ := updateModel(t, model, model.loadRegistryCmd()())
	view := failed.View().Content
	if !strings.Contains(view, "Process data is unavailable") || !strings.Contains(view, "[l] Retry") || strings.Contains(view, "Register") || strings.Contains(view, "private transport detail") {
		t.Fatalf("failure state: %s", view)
	}
	details, _ := updateModel(t, failed, key("d"))
	if !strings.Contains(details.View().Content, "private transport detail") {
		t.Fatal("startup diagnostic is inaccessible")
	}
	client.listErr = nil
	retrying, cmd := updateModel(t, details, key("l"))
	if cmd == nil || retrying.registry != registryLoading {
		t.Fatal("retry did not request registry")
	}
	empty, _ := updateModel(t, retrying, cmd())
	if empty.registry != registryEmpty || empty.notice != "" || empty.diagnostic != "" {
		t.Fatal("retry did not clear failure")
	}
	view = empty.View().Content
	if !strings.Contains(view, "No processes registered") || strings.Contains(view, "[d]") || strings.Contains(view, "Select") || strings.Contains(view, "Output") {
		t.Fatalf("empty state: %s", view)
	}
	help, _ := updateModel(t, empty, key("?"))
	for _, text := range []string{"dovik project add --id PROJECT --root", "dovik process add --project PROJECT --id PROCESS", "--command", "Replace PROJECT", "l to refresh"} {
		if !strings.Contains(help.View().Content, text) {
			t.Fatalf("setup missing %q", text)
		}
	}
}

func TestRegistryRefreshPreservesSelectionAndOutput(t *testing.T) {
	client := &fakeClient{listErr: errors.New("connection lost")}
	model := NewModel(context.Background(), client)
	model.loading = false
	model.registry = registryPopulated
	model.items = []processItem{{key: processKey{projectID: "demo", processID: "api"}}}
	model.statusKnown = true
	model.scroll = 2
	model.events = []supervision.OutputEvent{{Sequence: 1, Data: []byte("retained output")}}
	refreshing, cmd := updateModel(t, model, key("l"))
	failed, _ := updateModel(t, refreshing, cmd())
	if failed.selectedKey() != model.selectedKey() || failed.scroll != 2 || len(failed.events) != 1 || failed.registry != registryUnavailable {
		t.Fatal("failed refresh lost workspace")
	}
	if !strings.Contains(failed.View().Content, "UNAVAILABLE") {
		t.Fatal("stale state not labeled")
	}
	_, action := updateModel(t, failed, key("s"))
	if action != nil {
		t.Fatal("unavailable registry allowed an action")
	}
	retrying, _ := updateModel(t, failed, key("l"))
	recovered, _ := updateModel(t, retrying, registryLoadedMsg{items: model.items})
	if recovered.scroll != 2 || len(recovered.events) != 1 || recovered.registry != registryPopulated {
		t.Fatal("successful refresh discarded retained workspace")
	}
}

func TestStateLayoutsFitSupportedSizes(t *testing.T) {
	for _, size := range [][2]int{{64, 18}, {80, 24}, {120, 34}} {
		for _, state := range []registryOutcome{registryLoading, registryEmpty, registryUnavailable, registryPopulated} {
			model := Model{width: size[0], height: size[1], registry: state, presentation: newPresentation(false)}
			if state == registryPopulated {
				model.items = []processItem{{key: processKey{projectID: "demo", processID: "api"}}}
			}
			for _, help := range []bool{false, true} {
				model.showHelp = help
				view := model.View().Content
				if lipgloss.Width(view) != size[0] || lipgloss.Height(view) != size[1] {
					t.Fatalf("size %v state %v help %v: %dx%d", size, state, help, lipgloss.Width(view), lipgloss.Height(view))
				}
			}
			resized, _ := updateModel(t, model, tea.WindowSizeMsg{Width: 40, Height: 10})
			if !strings.Contains(resized.View().Content, "Terminal too small") {
				t.Fatal("missing minimum guidance")
			}
		}
	}
}
