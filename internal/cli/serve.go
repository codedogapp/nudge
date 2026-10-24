package cli

import (
	"context"
	"io"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/inbox"
)

// serve runs nudge serve until ctx ends.
func serve(ctx context.Context, args []string, stderr io.Writer) int {
	if len(args) != 0 {
		return misused(stderr)
	}

	if err := listenAndServe(ctx); err != nil {
		return failed(stderr, "serve", err)
	}

	return exitOK
}

func listenAndServe(ctx context.Context) error {
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
