package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/cli"
)

// socketIn points NUDGE_SOCKET at a socket in a missing folder of a new temporary folder, and
// returns its path. The path is short: macOS caps a socket's path at 104 bytes.
func socketIn(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "nudge")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	path := filepath.Join(dir, "nudge", "agent.sock")
	t.Setenv("NUDGE_SOCKET", path)

	return path
}

func TestRunPrintsTheUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"an unknown command", []string{"fly"}},
		{"serve with an argument", []string{"serve", "now"}},
		{"ask without a question", []string{"ask"}},
		{"ask with two questions", []string{"ask", "Merge?", "Now?"}},
		{"ask with a flag after the question", []string{"ask", "Merge?", "--option", "Yes"}},
		{"ask with an unknown flag", []string{"ask", "--colour", "red", "Merge?"}},
		{"ask for help", []string{"ask", "--help"}},
		{"mcp with an argument", []string{"mcp", "now"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			socketIn(t)

			var stdout, stderr strings.Builder

			if code := (cli.Program{}).Run(t.Context(), test.args, &stdout, &stderr); code != 2 {
				t.Errorf("Run(%q) = %d, want 2", test.args, code)
			}

			if !strings.Contains(stderr.String(), "usage: nudge serve\n") ||
				!strings.Contains(stderr.String(), " nudge mcp\n") || stdout.Len() != 0 {
				t.Errorf("stdout = %q, stderr = %q, want the usage on stderr",
					stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunNeedsTheSocketsPath(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"serve"}, "nudge serve: find the config folder: "},
		{[]string{"ask", "Merge?"}, "nudge ask: find the config folder: "},
	}

	for _, test := range tests {
		t.Run(test.args[0], func(t *testing.T) {
			configure(t)

			for _, name := range []string{"NUDGE_SOCKET", "HOME", "XDG_CONFIG_HOME"} {
				t.Setenv(name, "")
			}

			var stdout, stderr strings.Builder

			code := cli.Program{}.Run(t.Context(), test.args, &stdout, &stderr)
			if code != 1 || !strings.HasPrefix(stderr.String(), test.want) {
				t.Errorf("Run(%q) = %d, stderr = %q, want 1 and %q",
					test.args, code, stderr.String(), test.want)
			}
		})
	}
}
