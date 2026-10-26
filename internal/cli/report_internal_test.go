package cli

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/server"
	"github.com/codedogapp/nudge/internal/sessions"
)

func TestMCPReportsUnderTheCurrentProject(t *testing.T) {
	const private os.FileMode = 0o700

	// A short path from the system's temporary folder, which only the user can open.
	dir, err := os.MkdirTemp("", "nudge")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	socket := filepath.Join(dir, "nudge", "agent.sock")
	t.Setenv("NUDGE_SOCKET", socket)

	listener, err := agentdoor.Listen(t.Context(), socket)
	if err != nil {
		t.Fatal(err)
	}

	var (
		box  inbox.Inbox
		live sessions.Registry
	)

	go func() { _ = server.Serve(t.Context(), listener, agentdoor.Routes(&box, &live)) }()

	project := filepath.Join(t.TempDir(), "atlas")
	if err := os.Mkdir(project, private); err != nil {
		t.Fatal(err)
	}

	t.Chdir(project)

	serverEnd, agentEnd := net.Pipe()

	go func() {
		transport := &mcp.IOTransport{Reader: serverEnd, Writer: serverEnd}
		Program{}.serveMCP(t.Context(), nil, transport, os.Stderr)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	session, err := client.Connect(t.Context(),
		&mcp.IOTransport{Reader: agentEnd, Writer: agentEnd}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "report", Arguments: map[string]any{"note": "tests are green"},
	})
	if err != nil {
		t.Fatal(err)
	}

	listed := live.List()
	if len(listed) != 1 || listed[0].Project != atlas() || listed[0].Note != "tests are green" {
		t.Errorf("sessions = %+v, want atlas with its note", listed)
	}
}

func TestReportingFailsWithoutAConfigFolder(t *testing.T) {
	t.Setenv("NUDGE_SOCKET", "")
	t.Setenv("HOME", "")

	err := reporting(t.Context())(t.Context(), "hello")
	if err == nil {
		t.Error("report() = nil, want the error that stopped it")
	}
}

func atlas() string {
	host, _ := os.Hostname()
	short, _, _ := strings.Cut(host, ".")

	return "atlas (" + short + ")"
}
