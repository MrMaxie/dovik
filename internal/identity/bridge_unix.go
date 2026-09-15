//go:build linux || darwin

package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

// RunBridge runs inside the agent container. It carries no operator credential
// or administration capability; all incoming data remains untrusted by the host.
func RunBridge(ctx context.Context, input io.Reader, output io.Writer) error {
	endpoint := "/tmp/dovik-agent.sock"
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		return err
	}
	defer listener.Close()
	if err := os.Chmod(endpoint, 0600); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "dovik-agent-ready-v1"); err != nil {
		return err
	}
	go func() { <-ctx.Done(); listener.Close() }()
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)
	var mu sync.Mutex
	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go func() {
			defer connection.Close()
			mu.Lock()
			defer mu.Unlock()
			var request AgentRequest
			clientDecoder := json.NewDecoder(io.LimitReader(connection, 2*1024*1024))
			clientDecoder.DisallowUnknownFields()
			if err := clientDecoder.Decode(&request); err != nil {
				return
			}
			if err := encoder.Encode(request); err != nil {
				return
			}
			connected := true
			for {
				var frame Frame
				if err := decoder.Decode(&frame); err != nil {
					return
				}
				if connected {
					if err := json.NewEncoder(connection).Encode(frame); err != nil {
						connected = false
					}
				}
				if frame.ExitCode != nil || frame.Error != "" || request.Action != "execute" {
					return
				}
			}
		}()
	}
}
