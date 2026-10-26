package agentdoor

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// serverVar names the variable that holds the https URL of a nudge serve on another machine.
const serverVar = "NUDGE_SERVER"

// remote sends each request to the agent endpoints of a nudge serve on another machine.
type remote struct {
	server *url.URL
	base   http.RoundTripper
}

func (r remote) RoundTrip(req *http.Request) (*http.Response, error) {
	moved := req.Clone(req.Context())
	moved.URL.Scheme = r.server.Scheme
	moved.URL.Host = r.server.Host
	moved.URL.Path = r.server.Path + "/agent" + req.URL.Path
	moved.Host = r.server.Host

	return r.base.RoundTrip(moved)
}

// Connect returns the client for the agent door: one that reaches the nudge serve at NUDGE_SERVER
// through the tailnet, when it is set, else one that reaches the agent socket.
func Connect() (*http.Client, error) {
	if server := os.Getenv(serverVar); server != "" {
		return newRemote(server, http.DefaultTransport)
	}

	path, err := SocketPath()
	if err != nil {
		return nil, err
	}

	return NewClient(path), nil
}

// newRemote returns a client for the agent endpoints under server, an https URL, with no time
// limit, as a question waits for the user.
func newRemote(server string, base http.RoundTripper) (*http.Client, error) {
	parsed, err := url.Parse(server)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("%s must be an https URL: %w", serverVar,
			errors.Join(err, fmt.Errorf("got %q", server)))
	}

	return &http.Client{Transport: remote{server: parsed, base: base}}, nil
}
