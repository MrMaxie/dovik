package tui

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
	loading, _ := updateModel(t, opened, tea.KeyPressMsg{Code: 'c'})
	if loading.configureRequested {
		t.Fatal("configuration opened before project identity loaded")
	}
	opened, _ = updateModel(t, opened, identityLoadedMsg{snapshot: model.identities})
	view := ansi.Strip(opened.View().Content)
	for _, text := range []string{"Project identity", "Projects", "Example project", "github.com/owner/repo", "@example", "Read only", "1 of 8 operations", "Persona switching  Not allowed", "No isolated sessions", "[c] Configure"} {
		if !strings.Contains(view, text) {
			t.Fatalf("missing %q: %s", text, view)
		}
	}
	for _, unwanted := range []string{"Project 1/1:", "Protection: proxy-level", "Sessions: 0", "Allowed: read"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("identity view still contains raw diagnostic row %q: %s", unwanted, view)
		}
	}
	configured, _ := updateModel(t, opened, tea.KeyPressMsg{Code: 'c'})
	if !configured.configureRequested || configured.configureRoot != "/workspace" {
		t.Fatal("wrong configuration target")
	}
}
