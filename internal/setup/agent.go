package setup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	formula = "codedogapp/tap/nudge"
	// The tool call waits for the user's answer, so its timeout is a day.
	mcpTimeout = "86400000"
)

// AddToCopilot adds nudge's MCP server and its skill to Copilot CLI, and returns a line for each
// that says what it did. Both paths lie under Homebrew's prefix for the formula, which an
// upgrade keeps. An MCP server that Copilot CLI already has stays as it is.
func AddToCopilot(ctx context.Context, run Run) ([]string, error) {
	out, err := run(ctx, "brew", "--prefix", formula)
	if err != nil {
		return nil, fmt.Errorf("find the nudge formula: %w", err)
	}

	prefix := strings.TrimSpace(string(out))

	server, err := addServer(ctx, run, filepath.Join(prefix, "bin", "nudge"))
	if err != nil {
		return nil, err
	}

	skills := filepath.Join(prefix, "share", "nudge", "skills")
	if _, err := run(ctx, "copilot", "skill", "add", skills); err != nil {
		return nil, fmt.Errorf("add the skill: %w", err)
	}

	return []string{server, "copilot: added the skill from " + skills}, nil
}

// addServer adds the MCP server that runs bin mcp, unless Copilot CLI has a server of that name.
func addServer(ctx context.Context, run Run, bin string) (string, error) {
	_, err := run(ctx, "copilot", "mcp", "get", "nudge")
	if err == nil {
		return "copilot: kept the MCP server nudge", nil
	}

	// A server that isn't there makes `get` exit 1; any other error is a problem with the tool.
	if _, exited := errors.AsType[*exec.ExitError](err); !exited {
		return "", fmt.Errorf("look for the MCP server: %w", err)
	}

	args := []string{"mcp", "add", "--timeout", mcpTimeout, "nudge", "--", bin, "mcp"}
	if _, err := run(ctx, "copilot", args...); err != nil {
		return "", fmt.Errorf("add the MCP server: %w", err)
	}

	return "copilot: added the MCP server nudge", nil
}
