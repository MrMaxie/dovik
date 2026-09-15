package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWindowsNativePeerFixture(t *testing.T) {
	endpoint := os.Getenv("DOVIK_NATIVE_FIXTURE_ENDPOINT")
	if endpoint == "" {
		return
	}
	err := exerciseWindowsNativePeerFixture(endpoint, os.Getenv("DOVIK_NATIVE_FIXTURE_PRIVATE"))
	if resultPath := os.Getenv("DOVIK_NATIVE_FIXTURE_RESULT"); resultPath != "" {
		result := "ok"
		if err != nil {
			result = err.Error()
		}
		_ = os.WriteFile(resultPath, []byte(result), 0600)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func exerciseWindowsNativePeerFixture(endpoint, privatePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := DialAgent(ctx, endpoint)
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := json.NewEncoder(connection).Encode(AgentRequest{Version: 1, Action: "context"}); err != nil {
		return err
	}
	var frame Frame
	if err := json.NewDecoder(connection).Decode(&frame); err != nil {
		return err
	}
	if frame.Error != "" || frame.Context == nil || frame.Context.Persona.Account != "example" {
		return fmt.Errorf("wrong native identity context")
	}
	if _, err := os.ReadFile(privatePath); err == nil {
		return fmt.Errorf("agent read operator storage")
	}
	return nil
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
	resultPath := filepath.Join(projectRoot, "fixture-result.txt")
	environment := []string{
		"DOVIK_NATIVE_FIXTURE_ENDPOINT=" + session.Endpoint,
		"DOVIK_NATIVE_FIXTURE_PRIVATE=" + service.Store.path,
		"DOVIK_NATIVE_FIXTURE_RESULT=" + resultPath,
		"SystemRoot=" + os.Getenv("SystemRoot"),
		"TEMP=" + projectRoot,
		"TMP=" + projectRoot,
		"WINDIR=" + os.Getenv("WINDIR"),
	}
	if exitCode, err := runFixtureWithLogon(username, password, binary, projectRoot, environment); err != nil {
		t.Fatalf("separate Windows user: %v", err)
	} else if exitCode != 0 {
		result, _ := os.ReadFile(resultPath)
		t.Fatalf("separate Windows user exited with code %d: %s", exitCode, strings.TrimSpace(string(result)))
	}
}

var createProcessWithLogonW = windows.NewLazySystemDLL("advapi32.dll").NewProc("CreateProcessWithLogonW")

func runFixtureWithLogon(username, password, binary, directory string, environment []string) (uint32, error) {
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
	binaryPointer, err := windows.UTF16PtrFromString(binary)
	if err != nil {
		return 0, err
	}
	commandLine, err := windows.UTF16FromString(windows.ComposeCommandLine([]string{binary, "-test.run=^TestWindowsNativePeerFixture$"}))
	if err != nil {
		return 0, err
	}
	directoryPointer, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return 0, err
	}
	sort.Slice(environment, func(i, j int) bool {
		return strings.ToLower(environment[i]) < strings.ToLower(environment[j])
	})
	var environmentBlock []uint16
	for _, entry := range environment {
		environmentBlock = append(environmentBlock, utf16.Encode([]rune(entry))...)
		environmentBlock = append(environmentBlock, 0)
	}
	environmentBlock = append(environmentBlock, 0)
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var process windows.ProcessInformation
	result, _, callErr := createProcessWithLogonW.Call(
		uintptr(unsafe.Pointer(userPointer)),
		uintptr(unsafe.Pointer(domainPointer)),
		uintptr(unsafe.Pointer(passwordPointer)),
		0,
		uintptr(unsafe.Pointer(binaryPointer)),
		uintptr(unsafe.Pointer(&commandLine[0])),
		windows.CREATE_UNICODE_ENVIRONMENT,
		uintptr(unsafe.Pointer(&environmentBlock[0])),
		uintptr(unsafe.Pointer(directoryPointer)),
		uintptr(unsafe.Pointer(&startup)),
		uintptr(unsafe.Pointer(&process)),
	)
	if result == 0 {
		return 0, callErr
	}
	defer windows.CloseHandle(process.Process)
	defer windows.CloseHandle(process.Thread)
	event, err := windows.WaitForSingleObject(process.Process, 20_000)
	if err != nil {
		return 0, err
	}
	if event == uint32(windows.WAIT_TIMEOUT) {
		_ = windows.TerminateProcess(process.Process, 1)
		return 0, context.DeadlineExceeded
	}
	if event != windows.WAIT_OBJECT_0 {
		return 0, fmt.Errorf("unexpected process wait result %d", event)
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(process.Process, &exitCode); err != nil {
		return 0, err
	}
	return exitCode, nil
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
