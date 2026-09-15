package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/MrMaxie/dovik/internal/buildinfo"
	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/supervision"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(arguments []string, stdout io.Writer, stderr io.Writer) int {
	if len(arguments) == 1 && arguments[0] == "--version" {
		fmt.Fprintf(stdout, "dovikd %s\n", buildinfo.Version)
		return 0
	}
	if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h") {
		fmt.Fprintln(stdout, "Dovik daemon owns the local process registry, lifecycle, and control transport.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Usage: dovikd [--help] [--version]")
		return 0
	}
	if len(arguments) != 0 {
		fmt.Fprintf(stderr, "dovikd: unknown argument %q\n", arguments[0])
		return 2
	}
	if err := runDaemon(stdout); err != nil {
		fmt.Fprintf(stderr, "dovikd: %v\n", err)
		return 1
	}
	return 0
}

func runDaemon(stdout io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	registryPath, err := supervision.DefaultRegistryPath()
	if err != nil {
		return err
	}
	endpoint, err := control.DefaultEndpoint()
	if err != nil {
		return err
	}
	configuration, err := applyDaemonOverrides(daemonConfiguration{endpoint: endpoint, registryPath: registryPath})
	if err != nil {
		return err
	}
	store := supervision.NewFileRegistry(configuration.registryPath)
	registry, err := store.LoadAndReconcile(time.Now().UTC())
	if err != nil {
		return err
	}
	manager, err := supervision.NewLifecycleManager(registry, store, 1024*1024, 4096)
	if err != nil {
		return err
	}
	listener, err := control.ListenLocal(configuration.endpoint)
	if err != nil {
		return err
	}
	defer listener.Close()

	identityStore, err := identity.OpenStore(filepath.Join(filepath.Dir(configuration.registryPath), "identity", "state.json"))
	if err != nil {
		return err
	}
	identities := identity.NewService(identityStore)
	defer identities.Close()

	fmt.Fprintln(stdout, "Dovik daemon is running.")
	serveErr := control.NewServer(manager).WithIdentity(identities).Serve(ctx, listener)
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	shutdownErr := manager.Shutdown(shutdownContext)
	return errors.Join(serveErr, shutdownErr)
}
