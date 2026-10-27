// Package setup does the one-time work that nudge needs on a new machine: its settings file, and
// the agent's entries for the MCP server and the skill.
package setup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Run runs the command name with args, and returns its standard output. It is the seam through which
// every step reaches the outside tools.
type Run func(ctx context.Context, name string, args ...string) ([]byte, error)

// Exec is the Run that starts the real command. Its error names the command and holds what the
// command said on its standard error.
func Exec(ctx context.Context, name string, args ...string) ([]byte, error) {
	//nolint:gosec // G204: no shell runs; the callers pass fixed tool names and arguments.
	cmd := exec.CommandContext(ctx, name, args...)

	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if said := strings.TrimSpace(string(exitErr.Stderr)); said != "" {
			return nil, fmt.Errorf("%s: %w: %s", name, err, said)
		}
	}

	return nil, fmt.Errorf("%s: %w", name, err)
}
