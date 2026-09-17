package cli

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"time"

	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/operatorclient"
)

const whoamiTimeout = 400 * time.Millisecond

var dialAgentContext = func(ctx context.Context, endpoint string) (net.Conn, error) {
	return identity.DialAgent(ctx, endpoint)
}

type whoamiResult struct {
	State      string  `json:"state"`
	Configured *bool   `json:"configured"`
	Persona    *string `json:"persona"`
}

func runWhoAmI(ctx context.Context, client operatorclient.Client, stdout io.Writer, mode outputMode) error {
	statusCtx, cancel := context.WithTimeout(ctx, whoamiTimeout)
	defer cancel()

	result := resolveWhoAmI(statusCtx, client)
	if mode.json {
		return writeJSON(stdout, result)
	}
	value := "!"
	switch result.State {
	case "configured":
		value = *result.Persona
	case "unconfigured", "not_repository":
		value = "?"
	}
	_, err := io.WriteString(stdout, value+"\n")
	return err
}

func resolveWhoAmI(ctx context.Context, client operatorclient.Client) whoamiResult {
	if endpoint := os.Getenv("DOVIK_AGENT_ENDPOINT"); endpoint != "" {
		agentContext, err := callAgentContext(ctx, endpoint, "context", "")
		if err != nil || agentContext == nil || agentContext.Persona.Name == "" {
			return unavailableWhoAmI()
		}
		return configuredWhoAmI(agentContext.Persona.Name)
	}

	root, err := identity.GitRoot(ctx, ".")
	if err != nil {
		if ctx.Err() != nil {
			return unavailableWhoAmI()
		}
		configured := false
		return whoamiResult{State: "not_repository", Configured: &configured}
	}
	snapshot, err := client.Identity(ctx, identity.Request{Action: "get"})
	if err != nil {
		return unavailableWhoAmI()
	}
	project, err := snapshot.State.FindProject("", root)
	if err != nil {
		configured := false
		return whoamiResult{State: "unconfigured", Configured: &configured}
	}
	persona, err := snapshot.State.FindPersona(project.Persona)
	if err != nil || persona.Name == "" {
		return unavailableWhoAmI()
	}
	return configuredWhoAmI(persona.Name)
}

func configuredWhoAmI(name string) whoamiResult {
	configured := true
	return whoamiResult{State: "configured", Configured: &configured, Persona: &name}
}

func unavailableWhoAmI() whoamiResult {
	return whoamiResult{State: "unavailable"}
}

func callAgentContext(ctx context.Context, endpoint, action, persona string) (*identity.AgentContext, error) {
	connection, err := dialAgentContext(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if err := json.NewEncoder(connection).Encode(identity.AgentRequest{Version: identity.AgentProtocolVersion, Action: action, PersonaID: persona}); err != nil {
		return nil, err
	}
	var frame identity.Frame
	if err := json.NewDecoder(connection).Decode(&frame); err != nil {
		return nil, err
	}
	if frame.Error != "" {
		return nil, &agentContextError{message: frame.Error}
	}
	return frame.Context, nil
}

type agentContextError struct {
	message string
}

func (err *agentContextError) Error() string { return err.message }
