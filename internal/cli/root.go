package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"charm.land/huh/v2"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/identityui"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/MrMaxie/dovik/internal/terminalstyle"
	"github.com/MrMaxie/dovik/internal/tui"
)

type rootAction string

const (
	rootActionHelp      rootAction = "help"
	rootActionConfigure rootAction = "configure"
	rootActionTUI       rootAction = "tui"
	rootProbeTimeout               = 400 * time.Millisecond
	rootReadyTimeout               = 5 * time.Second
	rootRetryDelay                 = 100 * time.Millisecond
)

func runRootEntry(ctx context.Context, client operatorclient.Client, endpoint, defaultEndpoint string, input io.Reader, output io.Writer) error {
	root, configured := inspectRootProject(ctx, client)
	action, err := selectRootAction(ctx, root, configured, input, output)
	if rootMenuCanceled(err) {
		return nil
	}
	if err != nil {
		return err
	}

	switch action {
	case rootActionHelp:
		writeUsage(output)
		return nil
	case rootActionConfigure:
		launcher, err := resolveTUIDaemonLauncher(endpoint, defaultEndpoint)
		if err != nil {
			return err
		}
		if err := ensureIdentityDaemon(ctx, client, launcher); err != nil {
			return err
		}
		return identityui.Configure(ctx, client, root, input, output)
	case rootActionTUI:
		return runTUI(ctx, client, endpoint, defaultEndpoint, input, output)
	default:
		return fmt.Errorf("unknown root action")
	}
}

func rootMenuCanceled(err error) bool {
	return errors.Is(err, huh.ErrUserAborted)
}

func inspectRootProject(ctx context.Context, client operatorclient.Client) (string, *bool) {
	probeCtx, cancel := context.WithTimeout(ctx, rootProbeTimeout)
	defer cancel()
	root, err := identity.GitRoot(probeCtx, ".")
	if err != nil {
		return "", nil
	}
	snapshot, err := client.Identity(probeCtx, identity.Request{Action: "get"})
	if err != nil {
		return root, nil
	}
	_, err = snapshot.State.FindProject("", root)
	configured := err == nil
	return root, &configured
}

func selectRootAction(ctx context.Context, root string, configured *bool, input io.Reader, output io.Writer) (rootAction, error) {
	action := rootActionHelp
	options := rootMenuOptions(root, configured)
	field := huh.NewSelect[rootAction]().
		Title("What would you like to do?").
		Options(options...).
		Value(&action)
	err := terminalstyle.NewForm("Start", input, output, field).RunWithContext(ctx)
	return action, err
}

func rootMenuOptions(root string, configured *bool) []huh.Option[rootAction] {
	options := []huh.Option[rootAction]{huh.NewOption("Help", rootActionHelp)}
	if root != "" {
		label := "Configure project"
		if configured != nil && *configured {
			label = "Edit this repository"
		} else if configured != nil {
			label = "Configure this repository"
		}
		options = append(options, huh.NewOption(label, rootActionConfigure))
	}
	return append(options, huh.NewOption("Open TUI", rootActionTUI))
}

func ensureIdentityDaemon(ctx context.Context, client operatorclient.Client, launcher tui.DaemonLauncher) error {
	probe := func(probeCtx context.Context) error {
		_, err := client.Identity(probeCtx, identity.Request{Action: "get"})
		return err
	}
	probeCtx, cancel := context.WithTimeout(ctx, rootProbeTimeout)
	err := probe(probeCtx)
	cancel()
	if err == nil {
		return nil
	}
	if launcher == nil {
		return err
	}
	_, launchErr := launcher.Start()

	readyCtx, cancel := context.WithTimeout(ctx, rootReadyTimeout)
	defer cancel()
	var readinessErr error
	for {
		probeCtx, probeCancel := context.WithTimeout(readyCtx, rootProbeTimeout)
		readinessErr = probe(probeCtx)
		probeCancel()
		if readinessErr == nil {
			return nil
		}
		timer := time.NewTimer(rootRetryDelay)
		select {
		case <-readyCtx.Done():
			timer.Stop()
			if launchErr != nil {
				return errors.Join(launchErr, readinessErr)
			}
			return fmt.Errorf("daemon did not become available: %w", readinessErr)
		case <-timer.C:
		}
	}
}

func runTUI(ctx context.Context, client operatorclient.Client, endpoint, defaultEndpoint string, input io.Reader, output io.Writer) error {
	launcher, err := resolveTUIDaemonLauncher(endpoint, defaultEndpoint)
	if err != nil {
		return err
	}
	return tui.RunWithDaemonLauncher(ctx, client, launcher, input, output)
}

func rootEntryAllowed() bool {
	return os.Getenv("DOVIK_AGENT_ENDPOINT") == ""
}
