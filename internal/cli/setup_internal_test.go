package cli

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// fakeTools answers the setup steps' commands; the tool named failing fails.
func fakeTools(failing string) func(context.Context, string, ...string) ([]byte, error) {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		line := strings.Join(append([]string{name}, args...), " ")

		switch {
		case name == failing:
			return nil, errors.New("broken")
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

	job := setupJob{fakeTools(""), "https://nudge.tailnet.ts.net"}
	if code := job.runSetup(t.Context(), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit code = %d, stderr = %q", code, &stderr)
	}

	for _, want := range []string{"settings: wrote", "copilot: added the MCP", "/nudge"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout = %q, want %q", &stdout, want)
		}
	}
}

func TestSetupStopsAtTheFirstFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer

	job := setupJob{fakeTools("copilot"), "https://nudge.tailnet.ts.net"}
	if code := job.runSetup(t.Context(), &stdout, &stderr); code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}

	if !strings.Contains(stderr.String(), "nudge setup:") ||
		strings.Contains(stdout.String(), "added") {
		t.Errorf("stdout = %q, stderr = %q", &stdout, &stderr)
	}
}
