package vapid_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/vapid"
)

func TestAFileMakesTheKeyOnceAndKeepsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nudge", "vapid.key")

	first, err := vapid.NewFile(path).Load(t.Context())
	if err != nil || first.Private == "" || first.Public == "" {
		t.Fatalf("Load = %+v, %v", first, err)
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}

	second, err := vapid.NewFile(path).Load(t.Context())
	if err != nil || second != first {
		t.Errorf("second Load = %+v, %v; want the same key %+v", second, err, first)
	}
}

func TestAFileThatIsntAKeyIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vapid.key")
	_ = os.WriteFile(path, []byte("not a key\n"), 0o600)

	_, err := vapid.NewFile(path).Load(t.Context())
	if err == nil || !strings.Contains(err.Error(), "the VAPID key file "+path+" is invalid") {
		t.Errorf("err = %v", err)
	}

	if data, _ := os.ReadFile(filepath.Clean(path)); string(data) != "not a key\n" {
		t.Errorf("the file was replaced: %q", data)
	}
}

func TestAFileThatCantBeUsedIsAnError(t *testing.T) {
	dir := t.TempDir()

	if _, err := vapid.NewFile(dir).Load(t.Context()); err == nil {
		t.Error("a folder in place of the file gave no error")
	}

	blocker := filepath.Join(dir, "plain")
	_ = os.WriteFile(blocker, nil, 0o600)

	if _, err := vapid.NewFile(filepath.Join(blocker, "vapid.key")).Load(t.Context()); err == nil {
		t.Error("a file in place of the folder gave no error")
	}
}
