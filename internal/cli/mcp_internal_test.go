package cli

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// brokenTransport is a transport that can't connect.
type brokenTransport struct{}

func (brokenTransport) Connect(context.Context) (mcp.Connection, error) {
	return nil, errors.New("stdin is closed")
}

func TestServeMCPUntilTheAgentGoesAway(t *testing.T) {
	serverEnd, agentEnd := net.Pipe()
	exited := make(chan int, 1)

	var stderr bytes.Buffer

	go func() {
		transport := &mcp.IOTransport{Reader: serverEnd, Writer: serverEnd}
		exited <- serveMCP(t.Context(), nil, transport, &stderr)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	session, err := client.Connect(t.Context(),
		&mcp.IOTransport{Reader: agentEnd, Writer: agentEnd}, nil)
	if err != nil {
		t.Fatal(err)
	}

	info := session.InitializeResult().ServerInfo
	if info.Name != "nudge" || info.Version == "" || info.Version != version() {
		t.Errorf("server = %s %q, want nudge %q", info.Name, info.Version, version())
	}

	if err := agentEnd.Close(); err != nil {
		t.Fatal(err)
	}

	if code := <-exited; code != exitOK || stderr.Len() != 0 {
		t.Errorf("serveMCP = %d, stderr %q; want 0 and nothing", code, stderr.String())
	}
}

func TestServeMCPStopped(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	stop()

	serverEnd, _ := mcp.NewInMemoryTransports()

	var stderr bytes.Buffer
	if code := serveMCP(ctx, nil, serverEnd, &stderr); code != exitOK || stderr.Len() != 0 {
		t.Errorf("serveMCP = %d, stderr %q; want 0 and nothing", code, stderr.String())
	}
}

func TestServeMCPFails(t *testing.T) {
	var stderr bytes.Buffer

	code := serveMCP(t.Context(), nil, brokenTransport{}, &stderr)
	if code != exitError || stderr.String() != "nudge mcp: stdin is closed\n" {
		t.Errorf("serveMCP = %d, stderr %q; want 1 and the reason", code, stderr.String())
	}
}
