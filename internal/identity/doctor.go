package identity

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Check struct {
	Name   string `json:"name"`
	Ready  bool   `json:"ready"`
	Detail string `json:"detail,omitempty"`
}

// Credential locations come from the operator process, never from a checkout.
func credentialDirectories() []string {
	home, _ := os.UserHomeDir()
	config, _ := os.UserConfigDir()
	return []string{os.Getenv("GH_CONFIG_DIR"), filepath.Join(home, ".config", "gh"), filepath.Join(config, "gh")}
}

func (s *Service) nativeBoundary() error {
	for _, path := range []string{s.Store.Snapshot().GHPath} {
		if path != "" {
			if err := checkExecutableIntegrity(path); err != nil {
				return err
			}
		}
	}
	if err := checkPrivateDirectory(filepath.Dir(s.Store.path)); err != nil {
		return fmt.Errorf("identity storage is not private: %w", err)
	}
	for _, path := range credentialDirectories() {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("cannot inspect operator credential directory")
		}
		if err := checkPrivateDirectory(path); err != nil {
			return fmt.Errorf("restrict the original gh credential directory to the operator before starting native isolation: %w", err)
		}
		if err := filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
			if err != nil || entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("credential files must not redirect outside their private directory")
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) doctor(r Request) []Check {
	state := s.Store.Snapshot()
	_, ghErr := ValidateGH(state.GHPath)
	checks := []Check{{Name: "original-gh", Ready: ghErr == nil}}
	if ghErr != nil {
		checks[0].Detail = ghErr.Error()
	}
	boundaryErr := s.nativeBoundary()
	checks = append(checks, Check{Name: "private-operator-storage", Ready: boundaryErr == nil})
	if boundaryErr != nil {
		checks[len(checks)-1].Detail = boundaryErr.Error()
	}
	if r.Principal != "" {
		_, err := nativePrincipal(r.Principal)
		check := Check{Name: "native-account", Ready: err == nil}
		if err != nil {
			check.Detail = err.Error()
		}
		checks = append(checks, check)
	}
	if r.Backend == "docker" || r.Backend == "podman" {
		_, err := exec.LookPath(r.Backend)
		checks = append(checks, Check{Name: "container-runtime", Ready: err == nil, Detail: "Run an isolated session to verify the engine and image bridge."})
	}
	if r.SessionID != "" {
		s.mu.Lock()
		session, ok := s.sessions[r.SessionID]
		s.mu.Unlock()
		checks = append(checks, Check{Name: "authenticated-agent-channel", Ready: ok && session.Status == "active", Detail: "A native session becomes active after an authenticated connection from its assigned account."})
	}
	return checks
}
