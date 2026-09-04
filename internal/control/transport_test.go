package control

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestLocalTransportRoundTrip(t *testing.T) {
	endpoint, err := testEndpoint(t)
	if err != nil {
		t.Fatalf("testEndpoint() error = %v", err)
	}
	listener, err := ListenLocal(endpoint)
	if err != nil {
		t.Fatalf("ListenLocal() error = %v", err)
	}
	defer listener.Close()
	serverDone := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer connection.Close()
		content, readErr := io.ReadAll(io.LimitReader(connection, 4))
		if readErr == nil && string(content) != "ping" {
			readErr = fmt.Errorf("content = %q, want ping", content)
		}
		serverDone <- readErr
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, err := DialLocal(ctx, endpoint)
	if err != nil {
		t.Fatalf("DialLocal() error = %v", err)
	}
	if _, err := connection.Write([]byte("ping")); err != nil {
		connection.Close()
		t.Fatalf("Write() error = %v", err)
	}
	connection.Close()
	if err := <-serverDone; err != nil {
		t.Fatalf("server error = %v", err)
	}
}

func testEndpoint(t *testing.T) (string, error) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\dovik-test-%d-%d`, os.Getpid(), time.Now().UnixNano()), nil
	}
	return t.TempDir() + string(os.PathSeparator) + "state" + string(os.PathSeparator) + "control.sock", nil
}
