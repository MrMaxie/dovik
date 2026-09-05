package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MrMaxie/dovik/internal/control"
	"github.com/MrMaxie/dovik/internal/supervision"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dovikd: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
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

	fmt.Fprintln(os.Stdout, "Dovik daemon is running.")
	serveErr := control.NewServer(manager).Serve(ctx, listener)
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	shutdownErr := manager.Shutdown(shutdownContext)
	return errors.Join(serveErr, shutdownErr)
}
