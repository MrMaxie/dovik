package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintln(os.Stdout, "Dovik daemon foundation is running; process supervision is not implemented.")
	<-ctx.Done()
	fmt.Fprintln(os.Stdout, "Dovik daemon foundation stopped.")
}
