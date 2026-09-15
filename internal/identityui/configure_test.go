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
