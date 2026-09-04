//go:build windows

package control

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// DefaultEndpoint returns the current user's Named Pipe endpoint.
func DefaultEndpoint() (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("resolve current Windows user: %w", err)
	}
	sid := strings.ReplaceAll(user.User.Sid.String(), "-", "_")
	return `\\.\pipe\dovik-` + sid, nil
}

// ListenLocal opens a Named Pipe restricted to the current Windows user.
func ListenLocal(endpoint string) (net.Listener, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("resolve current Windows user: %w", err)
	}
	securityDescriptor := "D:P(A;;GA;;;" + user.User.Sid.String() + ")"
	listener, err := winio.ListenPipe(endpoint, &winio.PipeConfig{
		SecurityDescriptor: securityDescriptor,
		InputBufferSize:    64 * 1024,
		OutputBufferSize:   64 * 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("listen on Named Pipe: %w", err)
	}
	return listener, nil
}

// DialLocal connects to the local Named Pipe endpoint.
func DialLocal(ctx context.Context, endpoint string) (net.Conn, error) {
	connection, err := winio.DialPipeContext(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("connect to Named Pipe: %w", err)
	}
	return connection, nil
}
