//go:build windows

package identity

import (
	"context"
	"fmt"
	"net"
	"runtime"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func listenAgent(id, principal string) (net.Listener, string, error) {
	sid, err := windows.StringToSid(principal)
	if err != nil {
		return nil, "", fmt.Errorf("invalid native agent SID")
	}
	current, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, "", err
	}
	endpoint := `\\.\pipe\dovik-agent-` + id
	descriptor := "D:P(A;;GA;;;" + current.User.Sid.String() + ")(A;;GRGW;;;" + sid.String() + ")"
	listener, err := winio.ListenPipe(endpoint, &winio.PipeConfig{SecurityDescriptor: descriptor, InputBufferSize: 65536, OutputBufferSize: 65536})
	return listener, endpoint, err
}

var impersonatePipe = windows.NewLazySystemDLL("advapi32.dll").NewProc("ImpersonateNamedPipeClient")

func authenticateAgent(connection net.Conn, principal string, projectRoot string) error {
	pipe, ok := connection.(interface{ Fd() uintptr })
	if !ok {
		return fmt.Errorf("named pipe connection required")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	result, _, err := impersonatePipe.Call(pipe.Fd())
	if result == 0 {
		return fmt.Errorf("cannot authenticate native agent: %w", err)
	}
	defer func() {
		if err := windows.RevertToSelf(); err != nil {
			panic("cannot restore operator security context")
		}
	}()
	var token windows.Token
	if err := windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_QUERY, true, &token); err != nil {
		return fmt.Errorf("cannot inspect native agent token")
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil || user.User.Sid.String() != principal {
		return fmt.Errorf("native agent peer is not authorized")
	}
	groups, err := token.GetTokenGroups()
	if err != nil {
		return fmt.Errorf("cannot inspect native agent privileges")
	}
	for _, group := range groups.AllGroups() {
		switch group.Sid.String() {
		case "S-1-5-32-544", "S-1-5-32-551", "S-1-5-32-578":
			return fmt.Errorf("native agent must not be an administrator or privileged operator")
		}
	}
	if projectRoot != "" {
		path, err := windows.UTF16PtrFromString(projectRoot)
		if err != nil {
			return fmt.Errorf("invalid project root")
		}
		// FILE_WRITE_DATA is FILE_ADD_FILE for a directory handle.
		handle, err := windows.CreateFile(path, windows.FILE_LIST_DIRECTORY|windows.FILE_WRITE_DATA, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
		if err != nil {
			return fmt.Errorf("native account requires read and write access to the project")
		}
		windows.CloseHandle(handle)
	}
	for _, engine := range []string{`\\.\pipe\docker_engine`, `\\.\pipe\dockerDesktopLinuxEngine`} {
		path, _ := windows.UTF16PtrFromString(engine)
		handle, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
		if err == nil {
			windows.CloseHandle(handle)
			return fmt.Errorf("native account can access the container engine")
		}
		if err != windows.ERROR_ACCESS_DENIED && err != windows.ERROR_FILE_NOT_FOUND && err != windows.ERROR_PATH_NOT_FOUND {
			return fmt.Errorf("cannot verify that the native account is denied container engine access")
		}
	}
	return nil
}

func DialAgent(ctx context.Context, endpoint string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, endpoint)
}
