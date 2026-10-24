package agentdoor

import (
	"fmt"
	"os"
	"path/filepath"
)

// SocketPath is the agent socket's path: NUDGE_SOCKET when it's set, else nudge/agent.sock in the
// user's config folder (~/Library/Application Support on macOS).
func SocketPath() (string, error) {
	if path := os.Getenv("NUDGE_SOCKET"); path != "" {
		return path, nil
	}

	config, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find the config folder: %w", err)
	}

	return filepath.Join(config, "nudge", "agent.sock"), nil
}
