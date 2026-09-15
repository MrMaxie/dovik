package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

type Request struct {
	Action    string    `json:"action"`
	Persona   *Persona  `json:"persona,omitempty"`
	Project   *Project  `json:"project,omitempty"`
	ProjectID string    `json:"projectId,omitempty"`
	PersonaID string    `json:"personaId,omitempty"`
	SessionID string    `json:"sessionId,omitempty"`
	Policy    *Policy   `json:"policy,omitempty"`
	Path      string    `json:"path,omitempty"`
	Root      string    `json:"root,omitempty"`
	Backend   string    `json:"backend,omitempty"`
	Principal string    `json:"principal,omitempty"`
	Personas  []Persona `json:"personas,omitempty"`
}

type Session struct {
	Status    string `json:"status"`
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	PersonaID string `json:"personaId"`
	Backend   string `json:"backend"`
	Principal string `json:"principal,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
}

type Snapshot struct {
	State    State     `json:"state"`
	Sessions []Session `json:"sessions"`
	Checks   []Check   `json:"checks,omitempty"`
}

type ExecutionRequest struct {
	ProjectID  string     `json:"projectId,omitempty"`
	Root       string     `json:"root,omitempty"`
	SessionID  string     `json:"sessionId,omitempty"`
	Invocation Invocation `json:"invocation"`
}

type Service struct {
	Store     *Store
	Executor  Executor
	mu        sync.Mutex
	sessions  map[string]Session
	listeners map[string]net.Listener
	cancel    context.CancelFunc
	ctx       context.Context
}

func NewService(store *Store) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{Store: store, Executor: Executor{Directory: filepath.Dir(store.path)}, sessions: map[string]Session{}, listeners: map[string]net.Listener{}, ctx: ctx, cancel: cancel}
}

func (s *Service) Close() {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, listener := range s.listeners {
		_ = listener.Close()
	}
	s.sessions = map[string]Session{}
}

func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := Snapshot{State: s.Store.Snapshot(), Sessions: []Session{}}
	for _, session := range s.sessions {
		result.Sessions = append(result.Sessions, session)
	}
	slices.SortFunc(result.Sessions, func(a, b Session) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return result
}

func (s *Service) Call(ctx context.Context, r Request) (Snapshot, error) {
	var err error
	switch r.Action {
	case "doctor":
		snapshot := s.Snapshot()
		snapshot.Checks = s.doctor(r)
		return snapshot, nil
	case "persona.import":
		err = s.Store.Update(func(state *State) error {
			if len(r.Personas) == 0 {
				return fmt.Errorf("no personas to import")
			}
			for _, p := range r.Personas {
				if _, e := state.FindPersona(p.ID); e == nil {
					return fmt.Errorf("import would overwrite an existing persona; assign unused IDs first")
				}
				state.Personas = append(state.Personas, p)
			}
			return nil
		})
	case "configure":
		if r.Persona == nil || r.Project == nil {
			err = fmt.Errorf("persona and project are required")
			break
		}
		var path string
		path, err = ValidateGH(r.Path)
		if err != nil {
			break
		}
		var root string
		root, err = filepath.EvalSymlinks(r.Project.Root)
		if err != nil {
			break
		}
		project := *r.Project
		project.Root = root
		if info, e := os.Stat(root); e != nil || !info.IsDir() {
			return Snapshot{}, fmt.Errorf("project root must be an existing directory")
		}
		err = s.Store.Update(func(state *State) error {
			state.GHPath = path
			found := false
			for i, p := range state.Personas {
				if p.ID == r.Persona.ID {
					state.Personas[i] = *r.Persona
					found = true
					break
				}
			}
			if !found {
				state.Personas = append(state.Personas, *r.Persona)
			}
			found = false
			for i, p := range state.Projects {
				if p.ID == project.ID {
					state.Projects[i] = project
					found = true
					break
				}
			}
			if !found {
				state.Projects = append(state.Projects, project)
			}
			return nil
		})
	case "get":
	case "persona.put":
		if r.Persona == nil {
			err = fmt.Errorf("persona required")
		} else {
			err = s.Store.SetPersona(*r.Persona)
		}
	case "project.put":
		if r.Project == nil {
			err = fmt.Errorf("project required")
		} else {
			err = s.Store.SetProject(*r.Project)
		}
	case "gh.set":
		var path string
		path, err = ValidateGH(r.Path)
		if err == nil {
			err = s.Store.Update(func(state *State) error { state.GHPath = path; return nil })
		}
	case "policy.set":
		if r.Policy == nil {
			err = fmt.Errorf("policy required")
			break
		}
		err = s.Store.Update(func(state *State) error {
			for i, p := range state.Projects {
				if p.ID == r.ProjectID {
					state.Projects[i].Policy = *r.Policy
					return nil
				}
			}
			return fmt.Errorf("unknown project")
		})
	case "session.create":
		_, err = s.createSession(r)
	case "session.revoke":
		s.mu.Lock()
		if _, ok := s.sessions[r.SessionID]; !ok {
			err = fmt.Errorf("unknown session")
		} else {
			delete(s.sessions, r.SessionID)
			if listener := s.listeners[r.SessionID]; listener != nil {
				_ = listener.Close()
				delete(s.listeners, r.SessionID)
			}
		}
		s.mu.Unlock()
	case "session.persona":
		err = s.selectSessionPersona(r.SessionID, r.PersonaID)
	default:
		err = fmt.Errorf("unknown identity operation")
	}
	if err != nil {
		return Snapshot{}, err
	}
	return s.Snapshot(), nil
}

func (s *Service) createSession(r Request) (Session, error) {
	state := s.Store.Snapshot()
	p, err := state.FindProject(r.ProjectID, "")
	if err != nil {
		return Session{}, err
	}
	if !slices.Contains([]string{"proxy-level", "native", "docker", "podman"}, r.Backend) {
		return Session{}, fmt.Errorf("unknown session backend")
	}
	if p.Mode == "agent-isolation" && r.Backend == "proxy-level" {
		return Session{}, fmt.Errorf("project requires agent isolation")
	}
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return Session{}, err
	}
	session := Session{ID: hex.EncodeToString(nonce), ProjectID: p.ID, PersonaID: p.Persona, Backend: r.Backend, Principal: r.Principal, Status: "awaiting-agent"}
	if r.Backend == "proxy-level" {
		session.Status = "active"
	}
	if r.Backend == "native" {
		if err := s.nativeBoundary(); err != nil {
			return Session{}, err
		}
		principal, err := nativePrincipal(r.Principal)
		if err != nil {
			return Session{}, err
		}
		session.Principal = principal
		listener, endpoint, err := listenAgent(session.ID, principal)
		if err != nil {
			return Session{}, err
		}
		session.Endpoint = endpoint
		s.mu.Lock()
		s.sessions[session.ID] = session
		s.listeners[session.ID] = listener
		s.mu.Unlock()
		go s.serveAgent(session, listener)
	} else {
		s.mu.Lock()
		s.sessions[session.ID] = session
		s.mu.Unlock()
	}
	return session, nil
}

func (s *Service) selectSessionPersona(id, persona string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session is unavailable or revoked")
	}
	state := s.Store.Snapshot()
	p, err := state.FindProject(session.ProjectID, "")
	if err != nil {
		return err
	}
	if !p.Policy.Allows("persona") || (persona != p.Persona && !slices.Contains(p.Alternatives, persona)) {
		return fmt.Errorf("persona change is not permitted")
	}
	original, _ := state.FindPersona(p.Persona)
	alternative, err := state.FindPersona(persona)
	if err != nil || original.Host != alternative.Host {
		return fmt.Errorf("alternative persona must use the project host")
	}
	session.PersonaID = persona
	s.sessions[id] = session
	return nil
}

func (s *Service) activateSession(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[id]; ok {
		session.Status = "active"
		s.sessions[id] = session
	}
}

func (s *Service) Context(r ExecutionRequest) (Project, Persona, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.Store.Snapshot()
	if r.SessionID != "" {
		session, ok := s.sessions[r.SessionID]
		if !ok {
			return Project{}, Persona{}, fmt.Errorf("session is unavailable or revoked")
		}
		p, err := state.FindProject(session.ProjectID, "")
		if err != nil {
			return Project{}, Persona{}, err
		}
		if p.Mode == "agent-isolation" && session.Backend == "proxy-level" {
			return Project{}, Persona{}, fmt.Errorf("session no longer satisfies the project isolation policy")
		}
		if session.PersonaID != p.Persona && !slices.Contains(p.Alternatives, session.PersonaID) {
			return Project{}, Persona{}, fmt.Errorf("session persona is no longer permitted")
		}
		persona, err := state.FindPersona(session.PersonaID)
		return p, persona, err
	}
	p, err := state.FindProject(r.ProjectID, r.Root)
	if err != nil {
		return Project{}, Persona{}, err
	}
	if p.Mode == "agent-isolation" {
		return Project{}, Persona{}, fmt.Errorf("project requires an isolated session")
	}
	persona, err := state.FindPersona(p.Persona)
	return p, persona, err
}

func (s *Service) Execute(ctx context.Context, r ExecutionRequest, stdout, stderr io.Writer) (int, error) {
	project, persona, err := s.Context(r)
	if err != nil {
		return 1, err
	}
	if r.SessionID != "" && isGitCredentialInvocation(r.Invocation) {
		return 1, fmt.Errorf("Git credential helper is unavailable in isolated sessions")
	}
	state := s.Store.Snapshot()
	return s.Executor.Run(ctx, state.GHPath, project, persona, r.Invocation, stdout, stderr)
}
