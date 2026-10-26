package setup_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/setup"
)

const status = `{"Self":{"UserID":42,"DNSName":"machine.tailnet.ts.net."},
"User":{"42":{"LoginName":"user@example.com"},"7":{"LoginName":"other@example.com"}}}`

func answering(out string, err error) setup.Run {
	return func(context.Context, string, ...string) ([]byte, error) { return []byte(out), err }
}

func TestSettingsComeFromTheTailnetsStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nudge", "settings.json")

	line, err := setup.WriteSettings(t.Context(), answering(status, nil), path)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Clean(path))
	for _, want := range []string{`"login": "user@example.com"`, `"host": "machine.tailnet.ts.net"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("file = %s, want it to contain %s", data, want)
		}
	}

	if !strings.HasPrefix(line, "settings: wrote ") {
		t.Errorf("line = %q", line)
	}

	for _, p := range []string{filepath.Dir(path), path} {
		info, _ := os.Stat(p)
		if info.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s is open to others: %v", p, info.Mode())
		}
	}
}

func TestAnExistingFileStays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	_ = os.WriteFile(path, []byte("mine"), 0o600)

	line, err := setup.WriteSettings(t.Context(), answering("", errors.New("must not run")), path)
	if err != nil {
		t.Fatal(err)
	}

	if data, _ := os.ReadFile(filepath.Clean(path)); string(data) != "mine" {
		t.Errorf("file = %q, want it kept", data)
	}

	if !strings.HasPrefix(line, "settings: kept ") {
		t.Errorf("line = %q", line)
	}
}

func TestStatusProblemsAreErrors(t *testing.T) {
	tests := map[string]setup.Run{
		"the command fails": answering("", errors.New("not installed")),
		"not JSON":          answering("up", nil),
		"no login": answering(
			`{"Self":{"UserID":1,"DNSName":"a.b.ts.net."},"User":{}}`,
			nil,
		),
		"no host": answering(
			`{"Self":{"UserID":1},"User":{"1":{"LoginName":"a@b"}}}`,
			nil,
		),
	}

	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")

			if _, err := setup.WriteSettings(t.Context(), run, path); err == nil {
				t.Error("got no error")
			}

			if _, err := os.Stat(path); err == nil {
				t.Error("wrote a file anyway")
			}
		})
	}
}

func TestExecReturnsOutputAndNamesFailures(t *testing.T) {
	out, err := setup.Exec(t.Context(), "echo", "hi")
	if err != nil || string(out) != "hi\n" {
		t.Errorf("got %q, %v", out, err)
	}

	_, err = setup.Exec(t.Context(), "sh", "-c", "echo boom >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want the stderr", err)
	}

	if _, err = setup.Exec(t.Context(), "nudge-no-such-tool"); err == nil {
		t.Error("got no error for a missing tool")
	}
}
