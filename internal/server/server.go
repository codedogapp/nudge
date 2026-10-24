// Package server serves an HTTP handler on a listener until a context ends.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// Serve serves handler on listener until ctx ends, then returns once the requests have finished,
// or after 5 s. Each request's context ends with ctx, so a request that waits doesn't hold up the
// shutdown.
func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		BaseContext:       func(net.Listener) context.Context { return ctx },
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
		return err
	}

	if err := <-shutdown; err != nil {
		return fmt.Errorf("shut down: %w", err)
	}

	return nil
}
