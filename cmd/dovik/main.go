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
	code := cli.RunIO(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
