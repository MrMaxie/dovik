package supervision

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const registryFileName = "registry.json"

// FileRegistry persists the daemon registry in one private per-user file.
type FileRegistry struct {
	path string
}

// NewFileRegistry creates a registry store for an explicit file path.
func NewFileRegistry(path string) *FileRegistry {
	return &FileRegistry{path: path}
}

// DefaultRegistryPath returns the private per-user registry location.
func DefaultRegistryPath() (string, error) {
	var stateDirectory string
	var err error

	switch runtime.GOOS {
	case "windows":
		stateDirectory, err = os.UserCacheDir()
	case "linux":
		stateDirectory = os.Getenv("XDG_STATE_HOME")
		if stateDirectory != "" && !filepath.IsAbs(stateDirectory) {
			return "", fmt.Errorf("XDG_STATE_HOME must be absolute")
		}
		if stateDirectory == "" {
			var homeDirectory string
			homeDirectory, err = os.UserHomeDir()
			stateDirectory = filepath.Join(homeDirectory, ".local", "state")
		}
	default:
		stateDirectory, err = os.UserConfigDir()
	}
	if err != nil {
		return "", fmt.Errorf("resolve user state directory: %w", err)
	}

	return filepath.Join(stateDirectory, "dovik", registryFileName), nil
}

// Load reads and validates the current registry. A missing file represents an
// empty registry.
func (store *FileRegistry) Load() (*Registry, error) {
	if err := store.validatePath(); err != nil {
		return nil, err
	}
	content, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return NewRegistry(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	if err := verifyPrivateRegistryPermissions(store.path); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var document registryDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	if err := requireJSONEnd(decoder); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}

	registry, err := documentToRegistry(document)
	if err != nil {
		return nil, fmt.Errorf("validate registry: %w", err)
	}
	return registry, nil
}

// LoadAndReconcile loads the registry, reconciles interrupted runtimes, and
// persists the result before returning it to the daemon.
func (store *FileRegistry) LoadAndReconcile(at time.Time) (*Registry, error) {
	registry, err := store.Load()
	if err != nil {
		return nil, err
	}
	changed, err := registry.ReconcileAfterDaemonRestart(at)
	if err != nil {
		return nil, err
	}
	if changed {
		if err := store.Save(registry); err != nil {
			return nil, fmt.Errorf("persist reconciled registry: %w", err)
		}
	}
	return registry, nil
}

// Save validates and atomically replaces the registry file.
func (store *FileRegistry) Save(registry *Registry) error {
	if err := store.validatePath(); err != nil {
		return err
	}
	document, err := registryToDocument(registry)
	if err != nil {
		return fmt.Errorf("validate registry: %w", err)
	}
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode registry: %w", err)
	}
	content = append(content, '\n')
	if err := writeFileAtomically(store.path, content); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}
	return nil
}

func (store *FileRegistry) validatePath() error {
	if store == nil || !filepath.IsAbs(store.path) {
		return fmt.Errorf("registry path must be absolute")
	}
	return nil
}

func requireJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return fmt.Errorf("unexpected trailing JSON value")
	}
	return err
}

func writeFileAtomically(path string, content []byte) error {
	directory := filepath.Dir(path)
	if err := ensurePrivateDirectory(directory); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(directory, ".registry-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary registry: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict temporary registry permissions: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary registry: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("flush temporary registry: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary registry: %w", err)
	}
	if err := replaceFile(temporaryPath, path); err != nil {
		return fmt.Errorf("replace registry: %w", err)
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create registry directory: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect registry directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("registry directory path is not a directory")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("registry directory permissions %04o are not private", info.Mode().Perm())
	}
	return nil
}

func verifyPrivateRegistryPermissions(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directoryInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("inspect registry directory: %w", err)
	}
	if permissions := directoryInfo.Mode().Perm(); permissions&0o077 != 0 {
		return fmt.Errorf("registry directory permissions %04o are not private", permissions)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect registry file: %w", err)
	}
	if permissions := fileInfo.Mode().Perm(); permissions&0o077 != 0 {
		return fmt.Errorf("registry file permissions %04o are not private", permissions)
	}
	return nil
}
