package setup_test

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/setup"
)

// copilot is a fake brew and Copilot CLI that records each command line.
type copilot struct {
	ran       []string
	hasServer bool
	failing   string // a command-line prefix that fails, empty for none
	failWith  error
}

func (tool *copilot) run(_ context.Context, name string, args ...string) ([]byte, error) {
	line := strings.Join(append([]string{name}, args...), " ")
	tool.ran = append(tool.ran, line)

	switch {
	case tool.failing != "" && strings.HasPrefix(line, tool.failing):
		return nil, tool.failWith
	case strings.HasPrefix(line, "brew"):
		return []byte("/opt/homebrew/opt/nudge\n"), nil
	case strings.HasPrefix(line, "copilot mcp get") && !tool.hasServer:
		return nil, &exec.ExitError{}
	}

	return nil, nil
}

func TestCopilotGetsTheServerAndTheSkill(t *testing.T) {
	tool := &copilot{}

	lines, err := setup.AddToCopilot(t.Context(), tool.run)
	if err != nil || len(lines) != 2 {
		t.Fatalf("got %v, %v", lines, err)
	}

	for _, want := range []string{
		"copilot mcp add --timeout 86400000 nudge -- /opt/homebrew/opt/nudge/bin/nudge mcp",
		"copilot skill add /opt/homebrew/opt/nudge/share/nudge/skills",
	} {
		if !slices.Contains(tool.ran, want) {
			t.Errorf("ran %q, want %q among them", tool.ran, want)
		}
	}
}

func TestAnExistingServerStays(t *testing.T) {
	tool := &copilot{hasServer: true}

	lines, err := setup.AddToCopilot(t.Context(), tool.run)
	if err != nil || !strings.Contains(lines[0], "kept") {
		t.Fatalf("got %v, %v", lines, err)
	}

	for _, line := range tool.ran {
		if strings.HasPrefix(line, "copilot mcp add") {
			t.Errorf("ran %q on an existing server", line)
		}
	}
}

func TestEachFailureIsAnError(t *testing.T) {
	boom := errors.New("boom")
	tests := map[string]*copilot{
		"no brew":         {failing: "brew", failWith: boom},
		"no copilot":      {failing: "copilot mcp get", failWith: boom},
		"the add fails":   {failing: "copilot mcp add", failWith: boom},
		"the skill fails": {failing: "copilot skill add", failWith: boom},
	}

	for name, broken := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := setup.AddToCopilot(t.Context(), broken.run); !errors.Is(err, boom) {
				t.Errorf("err = %v, want it to wrap boom", err)
			}
		})
	}
}
