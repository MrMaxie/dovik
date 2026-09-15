package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"testing"
)

func fixtureService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "private", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := Persona{ID: "work", Name: "Work", GitName: "Example", GitEmail: "example@example.test", Host: "github.com", Account: "example"}
	if err := store.SetPersona(p); err != nil {
		t.Fatal(err)
	}
	if err := store.SetProject(Project{ID: "project", Name: "Project", Root: t.TempDir(), Repository: "owner/repo", Persona: "work", Mode: "proxy-level", Policy: Policy{Preset: "read-only"}, ProxyEnabled: true}); err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	t.Cleanup(service.Close)
	return service
}

func TestSessionsUseCurrentPolicyAndRevoke(t *testing.T) {
	s := fixtureService(t)
	ctx := context.Background()
	snapshot, err := s.Call(ctx, Request{Action: "session.create", ProjectID: "project", Backend: "proxy-level"})
	if err != nil {
		t.Fatal(err)
	}
	id := snapshot.Sessions[0].ID
	if _, err := s.Call(ctx, Request{Action: "policy.set", ProjectID: "project", Policy: &Policy{Preset: "maintain"}}); err != nil {
		t.Fatal(err)
	}
	p, _, err := s.Context(ExecutionRequest{SessionID: id})
	if err != nil || !p.Policy.Allows("merge") {
		t.Fatal("session has stale policy", err)
	}
	if _, err := s.Call(ctx, Request{Action: "session.revoke", SessionID: id}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Context(ExecutionRequest{SessionID: id}); err == nil {
		t.Fatal("revoked session authorized")
	}
	replacement := NewService(s.Store)
	defer replacement.Close()
	if _, _, err := replacement.Context(ExecutionRequest{SessionID: id}); err == nil {
		t.Fatal("session survived restart")
	}
}

func TestAgentProtocolHasNoAdministrativeOperations(t *testing.T) {
	s := fixtureService(t)
	snapshot, err := s.Call(context.Background(), Request{Action: "session.create", ProjectID: "project", Backend: "proxy-level"})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"persona.put", "project.put", "policy.set", "process.start", "session.create"} {
		server, client := net.Pipe()
		go func() { defer server.Close(); s.HandleAgent(context.Background(), snapshot.Sessions[0].ID, server) }()
		if err := json.NewEncoder(client).Encode(AgentRequest{Version: 1, Action: action}); err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(client)
		client.Close()
		if err != nil {
			t.Fatal(err)
		}
		var frame Frame
		if err := json.Unmarshal(data, &frame); err != nil || frame.Error == "" {
			t.Fatalf("%s authorized: %s", action, data)
		}
	}
}

func TestSessionCannotRequestGitCredential(t *testing.T) {
	s := fixtureService(t)
	snapshot, err := s.Call(context.Background(), Request{Action: "session.create", ProjectID: "project", Backend: "proxy-level"})
	if err != nil {
		t.Fatal(err)
	}
	request := ExecutionRequest{
		SessionID: snapshot.Sessions[0].ID,
		Invocation: Invocation{
			Arguments: []string{"auth", "git-credential", "get"},
			Input:     []byte("protocol=https\nhost=github.com\npath=owner/repo\n\n"),
		},
	}
	var output bytes.Buffer
	if _, err := s.Execute(context.Background(), request, &output, &output); err == nil {
		t.Fatal("isolated session received a Git credential")
	}
	if output.Len() != 0 {
		t.Fatal("isolated session received credential output")
	}
}
