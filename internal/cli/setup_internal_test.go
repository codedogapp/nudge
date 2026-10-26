package cli

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

const tailnetStatus = `{"Self":{"UserID":1,"DNSName":"machine.tailnet.ts.net."},
"User":{"1":{"LoginName":"user@example.com"}}}`

// fakeTools answers the setup steps' commands; the tool named failing fails.
func fakeTools(failing string) func(context.Context, string, ...string) ([]byte, error) {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		line := strings.Join(append([]string{name}, args...), " ")

		switch {
		case name == failing:
			return nil, errors.New("broken")
		case line == "tailscale status --json":
			return []byte(tailnetStatus), nil
		case name == "brew":
			return []byte("/opt/homebrew/opt/nudge\n"), nil
		case strings.HasPrefix(line, "copilot mcp get"):
			return nil, &exec.ExitError{}
		}

		return nil, nil
	}
}

func TestSetupRunsEveryStep(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer

	if code := runSetup(t.Context(), fakeTools(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit code = %d, stderr = %q", code, &stderr)
	}

	for _, want := range []string{"settings: wrote", "tailscale: serving", "copilot: added the MCP", "Home Screen"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout = %q, want %q", &stdout, want)
		}
	}
}

func TestSetupStopsAtTheFirstFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer

	if code := runSetup(
		t.Context(),
		fakeTools("tailscale"),
		&stdout,
		&stderr,
	); code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}

	if !strings.Contains(stderr.String(), "nudge setup:") ||
		strings.Contains(stdout.String(), "copilot") {
		t.Errorf("stdout = %q, stderr = %q", &stdout, &stderr)
	}
}
