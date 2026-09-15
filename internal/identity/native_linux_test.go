//go:build linux || darwin

package identity

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestNativePeerFixture(t *testing.T) {
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

func TestNativeSeparateUserContext(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root in a disposable native test environment")
	}
	accountName := os.Getenv("DOVIK_NATIVE_TEST_USER")
	if accountName == "" {
		accountName = "nobody"
	}
	account, err := user.Lookup(accountName)
	if err != nil {
		t.Skip("configured native test account unavailable")
	}
	uid, err := strconv.ParseUint(account.Uid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	gid, err := strconv.ParseUint(account.Gid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	service := fixtureService(t)
	projectRoot, err := os.MkdirTemp("", "dovik-native-project-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(projectRoot)
	if err := os.Chown(projectRoot, int(uid), int(gid)); err != nil {
		t.Fatal(err)
	}
	project := service.Store.Snapshot().Projects[0]
	project.Root = projectRoot
	if err := service.Store.SetProject(project); err != nil {
		t.Fatal(err)
	}
	session, err := service.createSession(Request{ProjectID: "project", Backend: "native", Principal: account.Username})
	if err != nil {
		t.Fatal(err)
	}
	own, err := net.Dial("unix", session.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewEncoder(own).Encode(AgentRequest{Version: 1, Action: "context"})
	_ = own.SetReadDeadline(time.Now().Add(time.Second))
	var frame Frame
	if json.NewDecoder(own).Decode(&frame) == nil && frame.Context != nil {
		t.Fatal("operator socket peer passed as agent")
	}
	own.Close()
	// Copy only this fixture binary into a traversable disposable directory.
	dir, err := os.MkdirTemp("", "dovik-native-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	self, _ := os.Executable()
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "fixture")
	if err := os.WriteFile(binary, data, 0755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, binary, "-test.run=^TestNativePeerFixture$")
	child.Env = []string{"DOVIK_NATIVE_FIXTURE_ENDPOINT=" + session.Endpoint, "DOVIK_NATIVE_FIXTURE_PRIVATE=" + service.Store.path}
	child.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}}
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("separate user: %v %s", err, output)
	}
}
