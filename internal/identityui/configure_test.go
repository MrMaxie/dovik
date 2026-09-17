package identityui

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/operatorclient"
)

type unusedClient struct{ operatorclient.Client }

func TestCanceledQuestionnaireCannotSave(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Configure(ctx, unusedClient{}, t.TempDir(), strings.NewReader(""), io.Discard); err == nil {
		t.Fatal("canceled questionnaire completed")
	}
}

func TestProxyLevelSummaryOmitsIsolationPermissions(t *testing.T) {
	project := identity.Project{Name: "Project", Repository: "owner/repo", Mode: "proxy-level"}
	persona := identity.Persona{Name: "Personal", Account: "maxie", Host: "github.com"}
	summary := configurationSummary(project, persona, []string{"read", "merge"}, false)
	if strings.Contains(summary, "Allowed:") || requiresIsolationPolicy(project.Mode) {
		t.Fatalf("proxy summary exposed isolation permissions: %q", summary)
	}
}

func TestAgentIsolationSummaryIncludesPermissions(t *testing.T) {
	project := identity.Project{Name: "Project", Repository: "owner/repo", Mode: "agent-isolation"}
	persona := identity.Persona{Name: "Work", Account: "work", Host: "github.com"}
	summary := configurationSummary(project, persona, []string{"read", "review"}, true)
	if !strings.Contains(summary, "Allowed: [read review]") || !requiresIsolationPolicy(project.Mode) {
		t.Fatalf("isolation summary = %q", summary)
	}
}

func TestProjectDetailsReuseConfiguredGHPath(t *testing.T) {
	project := identity.Project{ID: "project", Name: "Project", Repository: "owner/repo"}
	gh := `C:\scoop\shims\realgh.exe`

	for _, field := range projectDetailsFields(&project, &gh, false) {
		if field.GetKey() == "gh-path" {
			t.Fatal("configured global gh path was requested again")
		}
	}

	found := false
	for _, field := range projectDetailsFields(&project, &gh, true) {
		found = found || field.GetKey() == "gh-path"
	}
	if !found {
		t.Fatal("missing gh path field during initial global configuration")
	}
}
