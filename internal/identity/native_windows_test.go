package identity

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWindowsNativePeerFixture(t *testing.T) {
	endpoint := os.Getenv("DOVIK_NATIVE_FIXTURE_ENDPOINT")
	if endpoint == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := DialAgent(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := json.NewEncoder(connection).Encode(AgentRequest{Version: 1, Action: "context"}); err != nil {
		t.Fatal(err)
	}
	var frame Frame
	if err := json.NewDecoder(connection).Decode(&frame); err != nil {
		t.Fatal(err)
	}
	if frame.Error != "" || frame.Context == nil || frame.Context.Persona.Account != "example" {
		t.Fatal("wrong native identity context")
	}
	if _, err := os.ReadFile(os.Getenv("DOVIK_NATIVE_FIXTURE_PRIVATE")); err == nil {
		t.Fatal("agent read operator storage")
	}
}

func TestWindowsSeparateUserContext(t *testing.T) {
	username := os.Getenv("DOVIK_NATIVE_TEST_USERNAME")
	password := os.Getenv("DOVIK_NATIVE_TEST_PASSWORD")
	if username == "" || password == "" {
		t.Skip("requires a disposable local Windows account")
	}
	account, err := user.Lookup(username)
	if err != nil {
		t.Fatal(err)
	}
	service := fixtureService(t)
	projectRoot, err := os.MkdirTemp(os.Getenv("SystemDrive")+`\`, "dovik-native-project-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(projectRoot) })
	grant := exec.Command("icacls", projectRoot, "/grant", "*"+account.Uid+":(OI)(CI)M")
	if output, err := grant.CombinedOutput(); err != nil {
		t.Fatalf("grant project access: %v %s", err, output)
	}
	project := service.Store.Snapshot().Projects[0]
	project.Root = projectRoot
	if err := service.Store.SetProject(project); err != nil {
		t.Fatal(err)
	}
	session, err := service.createSession(Request{ProjectID: "project", Backend: "native", Principal: account.Uid})
	if err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(projectRoot, "native-fixture.exe")
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, data, 0o755); err != nil {
		t.Fatal(err)
	}
	token, err := logonFixtureUser(username, password)
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	for _, privilege := range []string{"SeAssignPrimaryTokenPrivilege", "SeIncreaseQuotaPrivilege"} {
		if err := enableProcessPrivilege(privilege); err != nil {
			t.Fatalf("enable %s: %v", privilege, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, binary, "-test.run=^TestWindowsNativePeerFixture$")
	child.Dir = projectRoot
	child.Env = append(os.Environ(), "DOVIK_NATIVE_FIXTURE_ENDPOINT="+session.Endpoint, "DOVIK_NATIVE_FIXTURE_PRIVATE="+service.Store.path)
	child.SysProcAttr = &syscall.SysProcAttr{Token: syscall.Token(token)}
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("separate Windows user: %v %s", err, output)
	}
}

func enableProcessPrivilege(name string) error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	namePointer, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	var luid windows.LUID
	if err := windows.LookupPrivilegeValue(nil, namePointer, &luid); err != nil {
		return err
	}
	privileges := windows.Tokenprivileges{PrivilegeCount: 1}
	privileges.Privileges[0] = windows.LUIDAndAttributes{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}
	return windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil)
}

var logonUserW = windows.NewLazySystemDLL("advapi32.dll").NewProc("LogonUserW")

func logonFixtureUser(username, password string) (windows.Token, error) {
	userPointer, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return 0, err
	}
	domainPointer, err := windows.UTF16PtrFromString(".")
	if err != nil {
		return 0, err
	}
	passwordPointer, err := windows.UTF16PtrFromString(password)
	if err != nil {
		return 0, err
	}
	var token windows.Token
	result, _, callErr := logonUserW.Call(
		uintptr(unsafe.Pointer(userPointer)),
		uintptr(unsafe.Pointer(domainPointer)),
		uintptr(unsafe.Pointer(passwordPointer)),
		2,
		0,
		uintptr(unsafe.Pointer(&token)),
	)
	if result == 0 {
		return 0, callErr
	}
	return token, nil
}

func TestNativeRejectsOperatorAccount(t *testing.T) {
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nativePrincipal(current.Uid); err == nil {
		t.Fatal("operator account accepted for isolation")
	}
}

func TestWindowsPipeChecksImpersonatedPeer(t *testing.T) {
	listener, endpoint, err := listenAgent("abcdef01234567890123456789", "S-1-5-32-545")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	result := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			result <- err
			return
		}
		defer connection.Close()
		var b [1]byte
		_, err = io.ReadFull(connection, b[:])
		if err == nil {
			err = authenticateAgent(connection, "S-1-5-32-545", "")
		}
		result <- err
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := DialAgent(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write([]byte("{")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("wrong Windows SID was authenticated")
		}
	case <-ctx.Done():
		t.Fatal("pipe authentication timed out")
	}
}

func TestWindowsPrivateDirectoryInspection(t *testing.T) {
	directory := t.TempDir()
	if err := protectPath(directory); err != nil {
		t.Fatal(err)
	}
	if err := checkPrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
}
