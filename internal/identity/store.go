package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu    sync.RWMutex
	path  string
	state State
}

func OpenStore(path string) (*Store, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("identity state path must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if err := protectPath(filepath.Dir(path)); err != nil {
		return nil, err
	}
	s := &Store{path: path, state: State{Version: 1, Personas: []Persona{}, Projects: []Project{}}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := protectPath(path); err != nil {
		return nil, err
	}
	if err := Decode(data, &s.state); err != nil {
		return nil, fmt.Errorf("invalid identity state")
	}
	if err := s.state.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Decode rejects unknown fields and trailing JSON values.
func Decode(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("unexpected trailing JSON")
	}
	return nil
}

func (s *Store) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneState(s.state)
}

func cloneState(state State) State {
	data, _ := json.Marshal(state)
	var result State
	_ = json.Unmarshal(data, &result)
	return result
}

func (s *Store) Update(change func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	if err := change(&next); err != nil {
		return err
	}
	if err := next.Validate(); err != nil {
		return err
	}
	for _, project := range next.Projects {
		if within(s.path, project.Root) || (next.GHPath != "" && within(next.GHPath, project.Root)) {
			return fmt.Errorf("project workspace must not contain identity state or the original gh executable")
		}
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(s.path), ".identity-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = protectPath(file.Name()); err == nil {
		_, err = file.Write(append(data, '\n'))
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := replaceState(file.Name(), s.path); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *Store) SetPersona(persona Persona) error {
	return s.Update(func(state *State) error {
		for i, p := range state.Personas {
			if p.ID == persona.ID {
				state.Personas[i] = persona
				return nil
			}
		}
		state.Personas = append(state.Personas, persona)
		return nil
	})
}

func (s *Store) SetProject(project Project) error {
	root, err := filepath.EvalSymlinks(project.Root)
	if err != nil {
		return fmt.Errorf("project root must exist")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("project root must be a directory")
	}
	project.Root = filepath.Clean(root)
	return s.Update(func(state *State) error {
		for i, p := range state.Projects {
			if p.ID == project.ID {
				state.Projects[i] = project
				return nil
			}
		}
		state.Projects = append(state.Projects, project)
		return nil
	})
}
