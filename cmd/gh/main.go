// gh is the optional, explicitly installed Dovik GitHub CLI proxy.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrMaxie/dovik/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	args := []string{}
	if endpoint := os.Getenv("DOVIK_ENDPOINT"); endpoint != "" {
		args = append(args, "--endpoint", endpoint)
	}
	args = append(args, "gh", "--")
	args = append(args, os.Args[1:]...)
	os.Exit(cli.RunIO(ctx, args, os.Stdin, os.Stdout, os.Stderr))
}
