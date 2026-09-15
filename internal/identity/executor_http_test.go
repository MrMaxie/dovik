package identity

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestConcurrentAccountsAndNoMutationRetry(t *testing.T) {
	var mu sync.Mutex
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer fixture-")
		if account != "alice" && account != "bob" {
			t.Error("unexpected test credential")
			w.WriteHeader(401)
			return
		}
		mu.Lock()
		requests[account+r.URL.Path]++
		mu.Unlock()
		switch r.URL.Path {
		case "/user":
			fmt.Fprint(w, account)
		case "/repos/owner/repo":
			fmt.Fprint(w, account+":owner/repo")
		case "/write":
			w.WriteHeader(503)
			fmt.Fprint(w, "fixture write failure")
		default:
			t.Error("unexpected destination")
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	directory := t.TempDir()
	source := filepath.Join(directory, "fixture.go")
	binary := filepath.Join(directory, "gh-original.exe")
	program := fmt.Sprintf(`package main
import("fmt";"os";"net/http";"io";"strings")
func main(){a:=os.Args[1:];if len(a)==1 {return};if a[0]=="auth" {if len(a)!=6||a[1]!="token"||a[2]!="--hostname"||a[4]!="--user" {os.Exit(97)};account:=a[5];if account=="missing" {fmt.Fprint(os.Stderr,"fixture-secret-missing");os.Exit(3)};fmt.Print("fixture-"+account);return}
path:="/repos/owner/repo";if a[0]=="api" {path="/user"} else {if !strings.Contains(strings.Join(a," "),"owner/repo") {os.Exit(88)};if a[1]=="comment" {path="/write"}}
request,_:=http.NewRequest("GET",%q+path,nil);request.Header.Set("Authorization","Bearer "+os.Getenv("GH_TOKEN"));response,err:=http.DefaultClient.Do(request);if err!=nil {os.Exit(89)};defer response.Body.Close();io.Copy(os.Stdout,response.Body);if response.StatusCode>=400 {os.Exit(42)}}`, server.URL)
	if err := os.WriteFile(source, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("go", "build", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, output)
	}
	executor := Executor{Directory: directory}
	project := Project{Repository: "owner/repo", Policy: Policy{Preset: "collaborate"}}
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		account := []string{"alice", "bob"}[i%2]
		group.Go(func() {
			var output, stderr bytes.Buffer
			code, err := executor.Run(context.Background(), binary, project, Persona{Host: "github.com", Account: account}, Invocation{Arguments: []string{"pr", "view", "1"}}, &output, &stderr)
			if err != nil || code != 0 || output.String() != account+":owner/repo" || stderr.Len() != 0 {
				t.Errorf("account %s: exit=%d error=%v output=%q", account, code, err, output.String())
			}
		})
	}
	group.Wait()
	var output, stderr bytes.Buffer
	code, err := executor.Run(context.Background(), binary, project, Persona{Host: "github.com", Account: "alice"}, Invocation{Arguments: []string{"pr", "comment", "1", "--body", "fixture"}}, &output, &stderr)
	if err != nil || code != 42 {
		t.Fatalf("write failure lost: %d %v", code, err)
	}
	if requests["alice/write"] != 1 || requests["alice/repos/owner/repo"] != 4 || requests["bob/repos/owner/repo"] != 4 {
		t.Fatal("incorrect account routing or mutation retry", requests)
	}
	output.Reset()
	stderr.Reset()
	code, err = executor.Run(context.Background(), binary, project, Persona{Host: "github.com", Account: "missing"}, Invocation{Arguments: []string{"pr", "view", "1"}}, &output, &stderr)
	if code == 0 || err == nil || strings.Contains(err.Error(), "fixture-secret") || output.Len() != 0 || stderr.Len() != 0 {
		t.Fatal("authentication failure exposed private output")
	}
}
