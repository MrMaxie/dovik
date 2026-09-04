package main

import (
	"context"
	"os"
	"time"

	"github.com/MrMaxie/dovik/internal/cli"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	code := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	cancel()
	os.Exit(code)
}
