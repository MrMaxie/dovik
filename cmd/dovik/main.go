package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/MrMaxie/dovik/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	args := os.Args[1:]
	if strings.EqualFold(filepath.Base(os.Args[0]), "gh") || strings.EqualFold(filepath.Base(os.Args[0]), "gh.exe") {
		args = append([]string{"gh", "--"}, args...)
	}
	code := cli.RunIO(ctx, args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
