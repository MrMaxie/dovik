// Package daemonlaunch starts the local daemon without making the caller its
// lifecycle owner.
package daemonlaunch

import (
	"fmt"
	"os"
	"path/filepath"
)

// Launcher starts one detached daemon process.
type Launcher struct {
	executable  string
	arguments   []string
	environment []string
}

// New creates a launcher for an exact daemon executable path.
func New(executable string, arguments, environment []string) (*Launcher, error) {
	if !filepath.IsAbs(executable) {
		return nil, fmt.Errorf("daemon executable path must be absolute")
	}
	return &Launcher{
		executable:  executable,
		arguments:   append([]string(nil), arguments...),
		environment: append([]string(nil), environment...),
	}, nil
}

// NewSibling resolves the daemon executable beside the running client.
func NewSibling() (*Launcher, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve client executable: %w", err)
	}
	extension := filepath.Ext(executable)
	daemon := filepath.Join(filepath.Dir(executable), "dovikd"+extension)
	info, err := os.Stat(daemon)
	if err != nil {
		return nil, fmt.Errorf("resolve sibling daemon executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("resolve sibling daemon executable: %s is not a regular file", daemon)
	}
	return New(daemon, nil, nil)
}

// Start launches and releases the daemon process immediately.
func (launcher *Launcher) Start() (int, error) {
	return startDetached(launcher.executable, launcher.arguments, launcher.environment)
}
