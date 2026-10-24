package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/phonedoor"
	"github.com/codedogapp/nudge/internal/server"
	"github.com/codedogapp/nudge/internal/subscriptions"
	"github.com/codedogapp/nudge/internal/tailscale"
	"github.com/codedogapp/nudge/internal/vapid"
)

// door is one of nudge serve's doors: a listener and the handler it serves.
type door struct {
	name     string
	listener net.Listener
	handler  http.Handler
}

// serve runs nudge serve until ctx ends.
func serve(ctx context.Context, args []string, stderr io.Writer) int {
	if len(args) != 0 {
		return misused(stderr)
	}

	if err := listenAndServe(ctx, slog.New(slog.NewTextHandler(stderr, nil))); err != nil {
		return failed(stderr, "serve", err)
	}

	return exitOK
}

// listenAndServe loads the VAPID key and the push subscriptions, opens the agent door and the
// phone door on one inbox, and serves them.
func listenAndServe(ctx context.Context, logger *slog.Logger) error {
	given, err := readSettings()
	if err != nil {
		return err
	}

	// The key loads in a moment; a stop that comes meanwhile ends serve right after.
	key, err := vapid.NewKeychain(given.security).Load(context.WithoutCancel(ctx))
	if err != nil {
		return err
	}

	devices, err := subscriptions.Open(given.subscriptions)
	if err != nil {
		return err
	}

	path, err := agentdoor.SocketPath()
	if err != nil {
		return err
	}

	agents, err := agentdoor.Listen(ctx, path)
	if err != nil {
		return err
	}

	var config net.ListenConfig

	phones, err := config.Listen(ctx, "tcp", given.listen)
	if err != nil {
		// Closing the agent door's listener removes its socket.
		_ = agents.Close()
		return fmt.Errorf("open the phone door: %w", err)
	}

	var box inbox.Inbox

	lock := tailscale.RequireLogin(given.login, given.host, logger)

	return serveDoors(ctx,
		door{"agent", agents, agentdoor.Routes(&box)},
		door{"phone", phones, lock(phonedoor.Routes(&box, key.Public, devices))},
	)
}

// serveDoors serves the doors until ctx ends or one of them fails, which stops the others too.
func serveDoors(ctx context.Context, doors ...door) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	errs := make([]error, len(doors))

	var serving sync.WaitGroup

	for i, open := range doors {
		serving.Go(func() {
			defer stop()

			if err := server.Serve(ctx, open.listener, open.handler); err != nil {
				errs[i] = fmt.Errorf("serve the %s door: %w", open.name, err)
			}
		})
	}

	serving.Wait()

	return errors.Join(errs...)
}
