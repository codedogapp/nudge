package setup_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/setup"
)

func TestSettingsHoldTheServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nudge", "settings.json")

	line, err := setup.WriteSettings(path, "https://nudge.tailnet.ts.net")
	if err != nil || !strings.HasPrefix(line, "settings: wrote ") {
		t.Fatalf("got %q, %v", line, err)
	}

	data, _ := os.ReadFile(filepath.Clean(path))
	if !strings.Contains(string(data), `"server": "https://nudge.tailnet.ts.net"`) {
		t.Errorf("file = %s", data)
	}

	for _, p := range []string{filepath.Dir(path), path} {
		if info, _ := os.Stat(p); info.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s is open to others: %v", p, info.Mode())
		}
	}
}

func TestAnExistingFileStays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	_ = os.WriteFile(path, []byte("mine"), 0o600)

	if line, err := setup.WriteSettings(path, "https://x.ts.net"); err != nil ||
		!strings.Contains(line, "kept") {
		t.Fatalf("got %q, %v", line, err)
	}

	if data, _ := os.ReadFile(filepath.Clean(path)); string(data) != "mine" {
		t.Errorf("file = %q", data)
	}
}

func TestTheServerMustBeAnHTTPSURL(t *testing.T) {
	for _, server := range []string{"", "nudge.ts.net", "http://nudge.ts.net", "https://"} {
		path := filepath.Join(t.TempDir(), "settings.json")
		if _, err := setup.WriteSettings(path, server); err == nil {
			t.Errorf("WriteSettings(%q) gave no error", server)
		}
	}
}

func TestTheFolderMustBeWritable(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	_ = os.WriteFile(file, nil, 0o600)

	if _, err := setup.WriteSettings(
		filepath.Join(file, "s.json"),
		"https://x.ts.net",
	); err == nil {
		t.Error("a path below a file gave no error")
	}

	if err := setup.WriteSkill(filepath.Join(file, "skills")); err == nil {
		t.Error("a skill folder below a file gave no error")
	}
}

func TestTheSkillIsWritten(t *testing.T) {
	folder := t.TempDir()

	for range 2 {
		if err := setup.WriteSkill(folder); err != nil {
			t.Fatal(err)
		}
	}

	data, err := os.ReadFile(filepath.Clean(filepath.Join(folder, "nudge", "SKILL.md")))
	if err != nil || !strings.Contains(string(data), "name: nudge") {
		t.Errorf("skill = %q, %v", data, err)
	}
}
