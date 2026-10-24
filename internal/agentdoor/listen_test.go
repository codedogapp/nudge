package agentdoor_test

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/agentdoor"
)

const (
	private  os.FileMode = 0o700
	readable os.FileMode = 0o750
	ownFile  os.FileMode = 0o600
)

// socketIn returns the path of a socket in a missing folder of a new temporary folder. The path
// is short: macOS caps a socket's path at 104 bytes, and t.TempDir's paths can be longer.
func socketIn(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "nudge")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	return filepath.Join(dir, "nudge", "agent.sock")
}

// listen calls Listen, and closes the listener at the end of the test.
func listen(t *testing.T, path string) net.Listener {
	t.Helper()

	listener, err := agentdoor.Listen(t.Context(), path)
	if err != nil {
		t.Fatalf("Listen() = %v, want nil", err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

func dial(t *testing.T, path string) {
	t.Helper()

	conn, err := (&net.Dialer{}).DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial the socket: %v", err)
	}

	_ = conn.Close()
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	return info.Mode().Perm()
}

func TestListenMakesAPrivateFolder(t *testing.T) {
	path := socketIn(t)

	listen(t, path)
	dial(t, path)

	if got := mode(t, filepath.Dir(path)); got != private {
		t.Errorf("the folder's mode = %v, want %v", got, private)
	}
}

func TestListenRefusesAnOpenFolder(t *testing.T) {
	path := socketIn(t)
	folder := filepath.Dir(path)

	if err := os.Mkdir(folder, readable); err != nil {
		t.Fatal(err)
	}

	_, err := agentdoor.Listen(t.Context(), path)
	if !errors.Is(err, agentdoor.ErrOpenFolder) ||
		!strings.HasSuffix(err.Error(), " has mode 0750") {
		t.Errorf("Listen() = %v, want %v ending in the mode 0750", err, agentdoor.ErrOpenFolder)
	}

	if got := mode(t, folder); got != readable {
		t.Errorf("the folder's mode = %v, want it left at %v", got, readable)
	}
}

func TestListenRefusesWhileAServerAnswers(t *testing.T) {
	path := socketIn(t)
	listen(t, path)

	if _, err := agentdoor.Listen(t.Context(), path); !errors.Is(err, agentdoor.ErrRunning) {
		t.Errorf("second Listen() = %v, want %v", err, agentdoor.ErrRunning)
	}

	dial(t, path)
}

// leaveStale leaves a socket at path that no server answers, as a server that crashed does.
func leaveStale(t *testing.T, path string) {
	t.Helper()

	stale, ok := listen(t, path).(*net.UnixListener)
	if !ok {
		t.Fatal("Listen() didn't return a *net.UnixListener")
	}

	stale.SetUnlinkOnClose(false)
	_ = stale.Close()
}

func TestListenReplacesAStaleSocket(t *testing.T) {
	path := socketIn(t)
	leaveStale(t, path)

	listen(t, path)
	dial(t, path)
}

func TestListenKeepsASocketItCantDial(t *testing.T) {
	path := socketIn(t)
	leaveStale(t, path)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := agentdoor.Listen(ctx, path); !errors.Is(err, context.Canceled) {
		t.Errorf("Listen() = %v, want %v", err, context.Canceled)
	}

	if info, err := os.Lstat(path); err != nil || info.Mode().Type() != fs.ModeSocket {
		t.Errorf("Lstat() = %v, %v, want the socket left in place", info, err)
	}
}

func TestListenLeavesOtherFiles(t *testing.T) {
	path := socketIn(t)

	if err := os.Mkdir(filepath.Dir(path), private); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("notes"), ownFile); err != nil {
		t.Fatal(err)
	}

	if _, err := agentdoor.Listen(t.Context(), path); !errors.Is(err, agentdoor.ErrNotSocket) {
		t.Errorf("Listen() = %v, want %v", err, agentdoor.ErrNotSocket)
	}

	if got, err := os.ReadFile(filepath.Clean(path)); err != nil || string(got) != "notes" {
		t.Errorf("the file = %q, %v, want it left as it was", got, err)
	}
}

func TestListenReportsWhatFailed(t *testing.T) {
	tests := []struct {
		name string
		path func(t *testing.T) string
		want string
	}{
		{
			"folder under a file",
			func(t *testing.T) string {
				file := filepath.Dir(socketIn(t))
				if err := os.WriteFile(file, nil, ownFile); err != nil {
					t.Fatal(err)
				}

				return filepath.Join(file, "agent.sock")
			},
			"make the socket's folder",
		},
		{
			"path too long",
			func(t *testing.T) string {
				return filepath.Join(filepath.Dir(socketIn(t)), strings.Repeat("a", 120))
			},
			"listen on the agent socket",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := agentdoor.Listen(t.Context(), test.path(t))
			if err == nil || !strings.HasPrefix(err.Error(), test.want) {
				t.Errorf("Listen() = %v, want an error that starts with %q", err, test.want)
			}
		})
	}
}
