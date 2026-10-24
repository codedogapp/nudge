package cli

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
)

func listen(t *testing.T) net.Listener {
	t.Helper()

	var config net.ListenConfig

	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	return listener
}

func TestServeDoorsStopsTheOthersWhenOneFails(t *testing.T) {
	closed := listen(t)
	_ = closed.Close()

	// serveDoors returns only once the phone door has stopped too.
	err := serveDoors(t.Context(),
		door{"phone", listen(t), http.NotFoundHandler()},
		door{"agent", closed, http.NotFoundHandler()},
	)

	if !errors.Is(err, net.ErrClosed) ||
		!strings.HasPrefix(err.Error(), "serve the agent door: ") ||
		strings.Contains(err.Error(), "phone") {
		t.Errorf("serveDoors() = %v, want only the agent door's %v", err, net.ErrClosed)
	}
}
