package server_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/codedogapp/nudge/internal/server"
)

type reply struct {
	status int
	body   string
	err    error
}

// listen opens a listener on a free loopback port.
func listen(t *testing.T) net.Listener {
	t.Helper()

	var config net.ListenConfig

	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() = %v, want nil", err)
	}

	return listener
}

func get(ctx context.Context, url string) reply {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return reply{err: err}
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return reply{err: err}
	}

	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()

	return reply{
		status: response.StatusCode,
		body:   string(body),
		err:    errors.Join(readErr, closeErr),
	}
}

func TestServeEndsTheRequestsWithItsContext(t *testing.T) {
	listener := listen(t)
	ctx, stop := context.WithCancel(t.Context())
	waiting := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(waiting)
		<-r.Context().Done()
		http.Error(w, "stopping", http.StatusServiceUnavailable)
	})
	served := make(chan error, 1)

	go func() { served <- server.Serve(ctx, listener, handler) }()

	done := make(chan reply, 1)

	go func() { done <- get(t.Context(), "http://"+listener.Addr().String()) }()

	<-waiting
	stop()

	if got := <-done; got.err != nil || got.status != http.StatusServiceUnavailable {
		t.Errorf("reply = %d %q, %v, want 503", got.status, got.body, got.err)
	}

	if err := <-served; err != nil {
		t.Errorf("Serve() = %v, want nil", err)
	}
}

func TestServeNeedsItsListener(t *testing.T) {
	listener := listen(t)
	_ = listener.Close()

	err := server.Serve(t.Context(), listener, http.NotFoundHandler())
	if !errors.Is(err, net.ErrClosed) {
		t.Errorf("Serve() on a closed listener = %v, want %v", err, net.ErrClosed)
	}
}
