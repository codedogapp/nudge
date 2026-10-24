package cli_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/cli"
)

func TestServeStopsWithItsContext(t *testing.T) {
	path := socketIn(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var stdout, stderr strings.Builder

	if code := cli.Run(ctx, []string{"serve"}, &stdout, &stderr); code != 0 {
		t.Errorf("Run(serve) = %d, stderr = %q, want 0", code, stderr.String())
	}

	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Lstat(the socket) = %v, want it gone", err)
	}
}

func TestServeReportsWhatFailed(t *testing.T) {
	const readable fs.FileMode = 0o750

	path := socketIn(t)
	if err := os.Mkdir(filepath.Dir(path), readable); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr strings.Builder

	code := cli.Run(t.Context(), []string{"serve"}, &stdout, &stderr)
	if want := "nudge serve: other users can open"; code != 1 ||
		!strings.HasPrefix(stderr.String(), want) {
		t.Errorf("Run(serve) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
	}
}
