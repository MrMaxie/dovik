// Package identity owns project personas and authorization for GitHub operations.
package identity

import (
	"fmt"
	"net/mail"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

type Persona struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	GitName  string `json:"gitName"`
	GitEmail string `json:"gitEmail"`
	Host     string `json:"host"`
	Account  string `json:"account"`
}

type Policy struct {
	Preset     string          `json:"preset"`
	Exceptions map[string]bool `json:"exceptions"`
}

var RepositoryOperations = []string{"read", "comment", "create", "edit", "close", "review", "merge", "actions"}

var Permissions = append(slices.Clone(RepositoryOperations), "persona")

func (p Policy) Allows(operation string) bool {
	if !slices.Contains(Permissions, operation) {
		return false
	}
	if allow, exists := p.Exceptions[operation]; exists {
		return allow
	}
	switch p.Preset {
	case "read-only":
		return operation == "read"
	case "collaborate":
		return slices.Contains([]string{"read", "comment", "create", "edit", "close", "review"}, operation)
	case "maintain":
		return operation != "persona"
	}
	return false
}

type Project struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Root         string   `json:"root"`
	Repository   string   `json:"repository"`
	Persona      string   `json:"persona"`
	Alternatives []string `json:"alternatives"`
	Mode         string   `json:"mode"`
	Policy       Policy   `json:"policy"`
	ProxyEnabled bool     `json:"proxyEnabled"`
}

type State struct {
	Version  int       `json:"version"`
	GHPath   string    `json:"ghPath"`
	Personas []Persona `json:"personas"`
	Projects []Project `json:"projects"`
}

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,99}$`)
var hostName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$`)

func (p Persona) Validate() error {
	if !identifier.MatchString(p.ID) || !identifier.MatchString(p.Account) || !hostName.MatchString(p.Host) || strings.Contains(p.Host, "..") {
		return fmt.Errorf("invalid persona ID, GitHub host, or account")
	}
	if !plainText(p.Name, 120) || !plainText(p.GitName, 120) {
		return fmt.Errorf("persona name and Git author are required")
	}
	address, err := mail.ParseAddress(p.GitEmail)
	if err != nil || address.Address != p.GitEmail {
		return fmt.Errorf("invalid Git email")
	}
	return nil
}

func plainText(value string, limit int) bool {
	if strings.TrimSpace(value) == "" || len(value) > limit {
		return false
	}
	return !strings.ContainsAny(value, "\x00\r\n\x1b")
}

func (s State) Validate() error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported identity state version")
	}
	if s.GHPath != "" && !filepath.IsAbs(s.GHPath) {
		return fmt.Errorf("original gh path must be absolute")
	}
	ids := map[string]bool{}
	for _, p := range s.Personas {
		if err := p.Validate(); err != nil {
			return err
		}
		if ids[p.ID] {
			return fmt.Errorf("duplicate persona ID")
		}
		ids[p.ID] = true
	}
	projects := map[string]bool{}
	roots := map[string]bool{}
	for _, p := range s.Projects {
		if !identifier.MatchString(p.ID) || projects[p.ID] {
			return fmt.Errorf("invalid or duplicate project ID")
		}
		projects[p.ID] = true
		if !plainText(p.Name, 120) || len(p.Description) > 4096 || strings.ContainsAny(p.Description, "\x00\x1b") {
			return fmt.Errorf("invalid project name or description")
		}
		if !filepath.IsAbs(p.Root) || filepath.Clean(p.Root) != p.Root {
			return fmt.Errorf("project root must be an absolute clean path")
		}
		root := pathKey(p.Root)
		if roots[root] {
			return fmt.Errorf("duplicate project root")
		}
		roots[root] = true
		parts := strings.Split(p.Repository, "/")
		if len(parts) != 2 || !identifier.MatchString(parts[0]) || !identifier.MatchString(parts[1]) {
			return fmt.Errorf("repository must be OWNER/REPO")
		}
		if !ids[p.Persona] {
			return fmt.Errorf("unknown project persona")
		}
		for _, alternative := range p.Alternatives {
			if !ids[alternative] {
				return fmt.Errorf("unknown alternative persona")
			}
			primary, _ := s.FindPersona(p.Persona)
			other, _ := s.FindPersona(alternative)
			if !strings.EqualFold(primary.Host, other.Host) {
				return fmt.Errorf("alternative personas must use the project host")
			}
		}
		if p.Mode != "proxy-level" && p.Mode != "agent-isolation" {
			return fmt.Errorf("unknown enforcement mode")
		}
		if !slices.Contains([]string{"read-only", "collaborate", "maintain"}, p.Policy.Preset) {
			return fmt.Errorf("select an explicit policy preset")
		}
		for operation := range p.Policy.Exceptions {
			if !slices.Contains(Permissions, operation) {
				return fmt.Errorf("unknown policy operation")
			}
		}
	}
	return nil
}

func (s State) FindPersona(id string) (Persona, error) {
	for _, p := range s.Personas {
		if p.ID == id {
			return p, nil
		}
	}
	return Persona{}, fmt.Errorf("persona is not configured")
}

func (s State) FindProject(id, root string) (Project, error) {
	for _, p := range s.Projects {
		if id != "" && p.ID == id {
			return p, nil
		}
		if id == "" && root != "" && pathKey(root) == pathKey(p.Root) {
			return p, nil
		}
	}
	return Project{}, fmt.Errorf("project configuration required; run dovik project configure")
}

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}
