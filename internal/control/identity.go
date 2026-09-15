package control

import (
	"context"
	"encoding/json"
	"io"
	"strconv"

	"github.com/MrMaxie/dovik/internal/identity"
)

func (client *Client) Identity(ctx context.Context, r identity.Request) (identity.Snapshot, error) {
	var result identity.Snapshot
	err := client.call(ctx, OperationIdentity, r, &result)
	return result, err
}

func (client *Client) ExecuteGH(ctx context.Context, r identity.ExecutionRequest, stdout, stderr io.Writer) (int, error) {
	return client.identityStream(ctx, OperationGHExecute, r, stdout, stderr)
}

func (client *Client) RunAgent(ctx context.Context, r identity.ContainerRequest, stdout, stderr io.Writer) (int, error) {
	return client.identityStream(ctx, OperationAgentRun, r, stdout, stderr)
}

func (client *Client) identityStream(ctx context.Context, operation Operation, r any, stdout, stderr io.Writer) (int, error) {
	connection, err := client.dial(ctx, client.endpoint)
	if err != nil {
		return 1, err
	}
	defer connection.Close()
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.Close()
		case <-finished:
		}
	}()
	payload, err := json.Marshal(r)
	if err != nil {
		return 1, err
	}
	request := Request{Version: ProtocolVersion, ID: strconv.FormatUint(client.nextID.Add(1), 10), Operation: operation, Payload: payload}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		return 1, err
	}
	return identity.ReadStream(connection, stdout, stderr)
}
