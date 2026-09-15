//go:build linux || darwin

package identity

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

func listenAgent(id, principal string) (net.Listener, string, error) {
	directory := filepath.Join(os.TempDir(), "dovik-agent-"+strconv.Itoa(os.Getuid()))
	if err := os.MkdirAll(directory, 0711); err != nil {
		return nil, "", err
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0711 {
		return nil, "", fmt.Errorf("native agent socket directory must be private and traversable (0711)")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Getuid()) {
		return nil, "", fmt.Errorf("agent socket directory is not owned by this operator")
	}
	endpoint := filepath.Join(directory, id[:24]+".sock")
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, "", err
	}
	// Peer credentials, not socket write access, are the authorization boundary.
	if err := os.Chmod(endpoint, 0666); err != nil {
		listener.Close()
		return nil, "", err
	}
	return listener, endpoint, nil
}

func authenticateAgent(connection net.Conn, principal string, projectRoot string) error {
	uid, err := peerUID(connection)
	if err != nil || strconv.Itoa(int(uid)) != principal {
		return fmt.Errorf("native agent peer is not authorized")
	}
	if projectRoot != "" {
		return nativeProjectAccess(projectRoot, principal)
	}
	return nil
}

func nativeProjectAccess(root, principal string) error {
	account, err := user.LookupId(principal)
	if err != nil {
		return err
	}
	groups, err := account.GroupIds()
	if err != nil {
		return err
	}
	for _, socket := range []string{"/var/run/docker.sock", "/run/podman/podman.sock", filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "podman", "podman.sock")} {
		info, e := os.Stat(socket)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return fmt.Errorf("cannot inspect container engine access")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("cannot inspect container engine ownership")
		}
		bits := info.Mode().Perm() & 7
		if strconv.Itoa(int(stat.Uid)) == principal {
			bits = (info.Mode().Perm() >> 6) & 7
		} else {
			for _, group := range groups {
				if group == strconv.Itoa(int(stat.Gid)) {
					bits = (info.Mode().Perm() >> 3) & 7
					break
				}
			}
		}
		if bits&2 != 0 {
			return fmt.Errorf("native agent account can access a container engine socket")
		}
	}
	for path := root; ; path = filepath.Dir(path) {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("cannot inspect project access")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("cannot inspect project ownership")
		}
		bits := info.Mode().Perm() & 7
		if strconv.Itoa(int(stat.Uid)) == principal {
			bits = (info.Mode().Perm() >> 6) & 7
		} else {
			for _, group := range groups {
				if group == strconv.Itoa(int(stat.Gid)) {
					bits = (info.Mode().Perm() >> 3) & 7
					break
				}
			}
		}
		required := os.FileMode(1)
		if path == root {
			required = 7
		}
		if bits&required != required {
			return fmt.Errorf("native account requires access to the project and its parent directories")
		}
		if filepath.Dir(path) == path {
			break
		}
	}
	return nil
}

func DialAgent(ctx context.Context, endpoint string) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, "unix", endpoint)
}
