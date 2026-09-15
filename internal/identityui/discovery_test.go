package identityui

import (
	"testing"

	"github.com/MrMaxie/dovik/internal/identity"
)

func TestPersonaChoicesPreserveSavedAndSuggestAccounts(t *testing.T) {
	saved := []identity.Persona{{ID: "personal", Host: "github.com", Account: "existing", Name: "Saved"}}
	accounts := []identity.GitHubAccount{{Host: "github.com", Account: "existing"}, {Host: "github.com", Account: "personal"}, {Host: "elsewhere.test", Account: "other"}}
	got := personaChoices(saved, accounts, identity.GitDefaults{Host: "github.com", Name: "Author", Email: "author@example.test"})
	if len(got) != 2 || got[0] != saved[0] || got[1].ID != "personal-2" || got[1].Account != "personal" || got[1].GitName != "Author" {
		t.Fatalf("%+v", got)
	}
	if len(saved) != 1 {
		t.Fatal("discovery changed saved personas")
	}
}
