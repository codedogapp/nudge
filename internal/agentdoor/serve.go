// Package agentdoor is nudge's door for agents: a Unix socket where each question waits for its
// answer.
package agentdoor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/codedogapp/nudge/internal/inbox"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// Serve serves the agent door on listener until ctx ends. Then the questions that wait get 503,
// and Serve returns once their requests have finished, or after 5 s.
func Serve(ctx context.Context, listener net.Listener, box *inbox.Inbox) error {
	server := &http.Server{
		Handler:           routes(box),
		ReadHeaderTimeout: readHeaderTimeout,
		// Requests end with ctx, so the questions that wait don't hold up the shutdown.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	shutdown := make(chan error, 1)

	stop := context.AfterFunc(ctx, func() {
		deadline, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		shutdown <- server.Shutdown(deadline)
	})
	defer stop()

	err := server.Serve(listener)
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve the agent door: %w", err)
	}

	if err := <-shutdown; err != nil {
		return fmt.Errorf("stop the agent door: %w", err)
	}

	return nil
}
