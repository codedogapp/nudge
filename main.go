// Command nudge hands a coding agent's questions to the user, wherever they are, and gives the
// answers back to the agent.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/inbox"
)

const usage = "usage: nudge serve"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) != 1 || args[0] != "serve" {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}

	if err := serve(); err != nil {
		fmt.Fprintln(os.Stderr, "nudge serve:", err)
		return 1
	}

	return 0
}

func serve() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	path, err := agentdoor.SocketPath()
	if err != nil {
		return err
	}

	listener, err := agentdoor.Listen(ctx, path)
	if err != nil {
		return err
	}

	var box inbox.Inbox

	return agentdoor.Serve(ctx, listener, &box)
}
