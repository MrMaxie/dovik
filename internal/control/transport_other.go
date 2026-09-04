//go:build !windows

package control

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// DefaultEndpoint returns the current user's Unix Domain Socket endpoint.
func DefaultEndpoint() (string, error) {
	runtimeDirectory := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDirectory != "" && !filepath.IsAbs(runtimeDirectory) {
		return "", fmt.Errorf("XDG_RUNTIME_DIR must be absolute")
	}
	if runtimeDirectory == "" {
		runtimeDirectory = filepath.Join(os.TempDir(), "dovik-"+strconv.Itoa(os.Getuid()))
	} else {
		runtimeDirectory = filepath.Join(runtimeDirectory, "dovik")
	}
	return filepath.Join(runtimeDirectory, "control.sock"), nil
}

// ListenLocal opens a private Unix Domain Socket.
func ListenLocal(endpoint string) (net.Listener, error) {
	if !filepath.IsAbs(endpoint) {
		return nil, fmt.Errorf("Unix Domain Socket path must be absolute")
	}
	directory := filepath.Dir(endpoint)
	if err := ensurePrivateSocketDirectory(directory); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(endpoint); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("control endpoint exists and is not a socket")
		}
		connection, dialErr := net.DialTimeout("unix", endpoint, 100*time.Millisecond)
		if dialErr == nil {
			connection.Close()
			return nil, fmt.Errorf("control endpoint is already active")
		}
		if err := os.Remove(endpoint); err != nil {
			return nil, fmt.Errorf("remove stale control socket: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect control endpoint: %w", err)
	}

	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, fmt.Errorf("listen on Unix Domain Socket: %w", err)
	}
	if err := os.Chmod(endpoint, 0o600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("restrict control socket permissions: %w", err)
	}
	return listener, nil
}

// DialLocal connects to the local Unix Domain Socket endpoint.
func DialLocal(ctx context.Context, endpoint string) (net.Conn, error) {
	dialer := net.Dialer{}
	connection, err := dialer.DialContext(ctx, "unix", endpoint)
	if err != nil {
		return nil, fmt.Errorf("connect to Unix Domain Socket: %w", err)
	}
	return connection, nil
}

func ensurePrivateSocketDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create control socket directory: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect control socket directory: %w", err)
	}
	if permissions := info.Mode().Perm(); permissions&0o077 != 0 {
		return fmt.Errorf("control socket directory permissions %04o are not private", permissions)
	}
	return nil
}
