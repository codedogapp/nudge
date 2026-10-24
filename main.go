// Command nudge hands a coding agent's questions to the user, wherever they are, and gives the
// answers back to the agent.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/codedogapp/nudge/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)

	stop()
	os.Exit(code)
}
