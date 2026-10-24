package agentdoor_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/server"
)

// pipes is a net.Listener whose connections are in-memory pipes. A server on it can run in a
// synctest bubble, where a goroutine that waits on a real socket isn't durably blocked.
type pipes struct {
	conns  chan net.Conn
	closed chan struct{}
	close  func()
}

func newPipes() *pipes {
	closed := make(chan struct{})

	return &pipes{
		conns:  make(chan net.Conn),
		closed: closed,
		close:  sync.OnceFunc(func() { close(closed) }),
	}
}

func (p *pipes) Accept() (net.Conn, error) {
	select {
	case conn := <-p.conns:
		return conn, nil
	case <-p.closed:
		return nil, net.ErrClosed
	}
}

func (p *pipes) Close() error {
	p.close()
	return nil
}

func (*pipes) Addr() net.Addr {
	return &net.UnixAddr{Name: "pipes", Net: "unix"}
}

// dial connects to the listener, as http.Transport.DialContext.
func (p *pipes) dial(ctx context.Context, _, _ string) (net.Conn, error) {
	client, accepted := net.Pipe()

	select {
	case p.conns <- accepted:
		return client, nil
	case <-p.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// door is the agent door, served on pipes in the test's bubble.
type door struct {
	box    inbox.Inbox
	client *http.Client
}

type reply struct {
	status int
	header http.Header
	body   string
	err    error
}

// serve serves the agent door until ctx ends. At the end of the test, it checks that server.Serve
// returned nil.
func serve(ctx context.Context, t *testing.T) *door {
	t.Helper()

	listener := newPipes()
	agent := &door{client: &http.Client{Transport: &http.Transport{DialContext: listener.dial}}}
	served := make(chan error, 1)

	go func() { served <- server.Serve(ctx, listener, agentdoor.Routes(&agent.box)) }()

	t.Cleanup(func() {
		if err := <-served; err != nil {
			t.Errorf("Serve() = %v, want nil", err)
		}

		agent.client.CloseIdleConnections()
	})

	return agent
}

func (d *door) do(ctx context.Context, method, path, body string) reply {
	url := "http://nudge" + path

	request, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	if err != nil {
		return reply{err: err}
	}

	response, err := d.client.Do(request)
	if err != nil {
		return reply{err: err}
	}

	got, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()

	return reply{
		status: response.StatusCode,
		header: response.Header,
		body:   string(got),
		err:    errors.Join(readErr, closeErr),
	}
}

// ask posts the merge question in a goroutine, and returns once the server has done all it can.
func (d *door) ask(ctx context.Context) <-chan reply {
	done := make(chan reply, 1)

	go func() { done <- d.do(ctx, http.MethodPost, "/questions", merge) }()

	synctest.Wait()

	return done
}
