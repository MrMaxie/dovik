package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const AgentProtocolVersion = 1

type AgentRequest struct {
	Version    int        `json:"version"`
	Action     string     `json:"action"`
	Invocation Invocation `json:"invocation,omitempty"`
	PersonaID  string     `json:"personaId,omitempty"`
}

type Frame struct {
	Stream   string        `json:"stream,omitempty"`
	Data     []byte        `json:"data,omitempty"`
	ExitCode *int          `json:"exitCode,omitempty"`
	Error    string        `json:"error,omitempty"`
	Context  *AgentContext `json:"context,omitempty"`
}
type AgentContext struct {
	Project Project `json:"project"`
	Persona Persona `json:"persona"`
}

type frameWriter struct {
	encoder *json.Encoder
	mu      *sync.Mutex
	stream  string
}

func (w frameWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for offset := 0; offset < len(p); {
		end := min(offset+32*1024, len(p))
		if err := w.encoder.Encode(Frame{Stream: w.stream, Data: p[offset:end]}); err != nil {
			return 0, err
		}
		offset = end
	}
	return len(p), nil
}

func (s *Service) Stream(ctx context.Context, r ExecutionRequest, writer io.Writer) {
	encoder := json.NewEncoder(writer)
	var mu sync.Mutex
	code, err := s.Execute(ctx, r, frameWriter{encoder, &mu, "stdout"}, frameWriter{encoder, &mu, "stderr"})
	frame := Frame{ExitCode: &code}
	if err != nil {
		frame.Error = err.Error()
	}
	_ = encoder.Encode(frame)
}

func (s *Service) StreamContainer(ctx context.Context, r ContainerRequest, writer io.Writer) {
	encoder := json.NewEncoder(writer)
	var mu sync.Mutex
	code, err := s.RunContainer(ctx, r, frameWriter{encoder, &mu, "stdout"}, frameWriter{encoder, &mu, "stderr"})
	frame := Frame{ExitCode: &code}
	if err != nil {
		frame.Error = err.Error()
	}
	_ = encoder.Encode(frame)
}

func ReadStream(reader io.Reader, stdout, stderr io.Writer) (int, error) {
	decoder := json.NewDecoder(reader)
	for {
		var frame Frame
		if err := decoder.Decode(&frame); err != nil {
			return 1, fmt.Errorf("gh execution channel closed")
		}
		if frame.ExitCode != nil {
			if frame.Error != "" {
				return *frame.ExitCode, fmt.Errorf("%s", frame.Error)
			}
			return *frame.ExitCode, nil
		}
		var writer io.Writer
		switch frame.Stream {
		case "stdout":
			writer = stdout
		case "stderr":
			writer = stderr
		default:
			return 1, fmt.Errorf("invalid gh output frame")
		}
		if _, err := writer.Write(frame.Data); err != nil {
			return 1, err
		}
	}
}

func (s *Service) serveAgent(session Session, listener net.Listener) {
	for {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer connection.Close()
			_ = connection.SetReadDeadline(time.Now().Add(10 * time.Second))
			// Windows impersonation authenticates the sender of the last pipe read.
			first := make([]byte, 1)
			if _, err := io.ReadFull(connection, first); err != nil {
				return
			}
			project, err := s.Store.Snapshot().FindProject(session.ProjectID, "")
			if err != nil {
				return
			}
			if err := authenticateAgent(connection, session.Principal, project.Root); err != nil {
				return
			}
			if err := s.nativeBoundary(); err != nil {
				return
			}
			s.activateSession(session.ID)
			s.HandleAgent(s.ctx, session.ID, readWriter{Reader: io.MultiReader(bytes.NewReader(first), connection), Writer: connection})
		}()
	}
}

type readWriter struct {
	io.Reader
	io.Writer
}

// HandleAgent exposes no administrative operations and binds every request to
// the session selected by the trusted transport, never to client-supplied IDs.
func (s *Service) HandleAgent(ctx context.Context, sessionID string, connection io.ReadWriter) {
	var request AgentRequest
	decoder := json.NewDecoder(io.LimitReader(connection, 2*1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.Version != AgentProtocolVersion {
		_ = json.NewEncoder(connection).Encode(Frame{Error: "invalid agent request"})
		return
	}
	r := ExecutionRequest{SessionID: sessionID, Invocation: request.Invocation}
	if request.Action == "execute" {
		s.Stream(ctx, r, connection)
		return
	}
	var frame Frame
	switch request.Action {
	case "context":
		p, persona, err := s.Context(r)
		if err != nil {
			frame.Error = err.Error()
		} else {
			p.Root = ""
			frame.Context = &AgentContext{Project: p, Persona: persona}
		}
	case "persona":
		if err := s.selectSessionPersona(sessionID, request.PersonaID); err != nil {
			frame.Error = err.Error()
		}
	default:
		frame.Error = "agent operation is not permitted"
	}
	_ = json.NewEncoder(connection).Encode(frame)
}
