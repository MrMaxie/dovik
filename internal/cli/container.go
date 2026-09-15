package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/operatorclient"
)

func runContainerSession(ctx context.Context, client operatorclient.Client, args []string, stdin io.Reader, stdout, stderr io.Writer, mode outputMode) (int, error) {
	if os.Getenv("DOVIK_AGENT_ENDPOINT") != "" {
		return 1, fmt.Errorf("start agent sessions from the operator terminal")
	}
	if mode.json {
		return 2, fmt.Errorf("session run forwards agent output; --json is not supported")
	}
	flags := newFlagSet("session run", stderr, mode)
	project := flags.String("project", "", "project ID")
	backend := flags.String("backend", "docker", "docker or podman")
	image := flags.String("image", "", "prepared agent image")
	inputFile := flags.String("input", "", "input file, or - for stdin")
	var resources stringList
	flags.Var(&resources, "resource", "read-only SOURCE=TARGET resource")
	if err := flags.Parse(args); err != nil {
		return 2, err
	}
	request := identity.ContainerRequest{ProjectID: *project, Backend: *backend, Image: *image, Command: flags.Args()}
	for _, resource := range resources {
		source, target, ok := strings.Cut(resource, "=")
		if !ok {
			return 2, fmt.Errorf("resource must be SOURCE=TARGET")
		}
		request.Resources = append(request.Resources, identity.Resource{Source: source, Target: target})
	}
	if *inputFile != "" {
		reader := stdin
		if *inputFile != "-" {
			file, err := os.Open(*inputFile)
			if err != nil {
				return 1, err
			}
			defer file.Close()
			reader = file
		}
		data, err := io.ReadAll(io.LimitReader(reader, 1024*1024+1))
		if err != nil || len(data) > 1024*1024 {
			return 1, fmt.Errorf("agent input exceeds 1 MiB or is unreadable")
		}
		request.Input = data
	}
	return client.RunAgent(ctx, request, stdout, stderr)
}
