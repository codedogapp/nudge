package setup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// The tool call waits for the user's answer, so its timeout is a day.
const mcpTimeout = "86400000"

// AddToCopilot adds nudge's MCP server, which runs bin, and the skill in skills to Copilot CLI,
// and returns a line for each that says what it did. An MCP server that Copilot CLI already has
// stays as it is.
func AddToCopilot(ctx context.Context, run Run, bin, skills string) ([]string, error) {
	server, err := addServer(ctx, run, bin)
	if err != nil {
		return nil, err
	}

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
