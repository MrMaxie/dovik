package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorePersistsAndRejectsInvalidUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "state.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	p := Persona{ID: "work", Name: "Work", GitName: "Example", GitEmail: "example@example.test", Host: "github.com", Account: "example"}
	if err := store.SetPersona(p); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	p.Account = "--evil"
	if store.SetPersona(p) == nil {
		t.Fatal("invalid persona accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed update changed disk")
	}
	restored, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Snapshot().Personas) != 1 {
		t.Fatal("persona not restored")
	}
	snapshot := restored.Snapshot()
	snapshot.Personas[0].Name = "changed"
	if restored.Snapshot().Personas[0].Name == "changed" {
		t.Fatal("snapshot aliases live state")
	}
}

func TestCorruptStateIsNotReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"version":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(path); err == nil {
		t.Fatal("accepted corrupt state")
	}
	content, _ := os.ReadFile(path)
	if string(content) != `{"version":2}` {
		t.Fatal("corruption overwritten")
	}
}
