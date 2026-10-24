package agentdoor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codedogapp/nudge/internal/agentdoor"
)

func TestSocketPathFollowsNudgeSocket(t *testing.T) {
	t.Setenv("NUDGE_SOCKET", "/tmp/elsewhere.sock")

	if got, err := agentdoor.SocketPath(); err != nil || got != "/tmp/elsewhere.sock" {
		t.Errorf("SocketPath() = %q, %v, want /tmp/elsewhere.sock", got, err)
	}
}

func TestSocketPathDefaultsToTheConfigFolder(t *testing.T) {
	t.Setenv("NUDGE_SOCKET", "")

	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(config, "nudge", "agent.sock")
	if got, err := agentdoor.SocketPath(); err != nil || got != want {
		t.Errorf("SocketPath() = %q, %v, want %q", got, err, want)
	}
}

func TestSocketPathNeedsAHome(t *testing.T) {
	t.Setenv("NUDGE_SOCKET", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	if got, err := agentdoor.SocketPath(); err == nil {
		t.Errorf("SocketPath() without a home = %q, want an error", got)
	}
}
