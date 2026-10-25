package cli_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/cli"
)

// configure sets nudge serve's settings, with the phone door on a free loopback port and the
// subscriptions in a file of the test's own, gives it a home of its own, with no settings file,
// and a Keychain that holds a VAPID key.
func configure(t *testing.T) {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("NUDGE_LOGIN", "user@example.com")
	t.Setenv("NUDGE_HOST", "machine.tailnet.ts.net")
	t.Setenv("NUDGE_LISTEN", "127.0.0.1:0")
	t.Setenv("NUDGE_SUBSCRIPTIONS", filepath.Join(t.TempDir(), "subscriptions.json"))
	standIn(t, "keychain")
}

// settingsPath is the path of nudge serve's settings file in the test's home.
func settingsPath(t *testing.T) string {
	t.Helper()

	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	return filepath.Join(config, "nudge", "settings.json")
}

// standIn points NUDGE_SECURITY at the stand-in security tool in testdata/dir.
func standIn(t *testing.T, dir string) {
	t.Helper()

	abs, err := filepath.Abs(filepath.Join("testdata", dir, "security"))
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("NUDGE_SECURITY", abs)
}

func assertGone(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Lstat(the socket) = %v, want it gone", err)
	}
}

func TestServeStopsWithItsContext(t *testing.T) {
	path := socketIn(t)
	configure(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var stdout, stderr strings.Builder

	if code := cli.Run(ctx, []string{"serve"}, &stdout, &stderr); code != 0 {
		t.Errorf("Run(serve) = %d, stderr = %q, want 0", code, stderr.String())
	}

	assertGone(t, path)
}

func TestServeReportsWhatFailed(t *testing.T) {
	const readable fs.FileMode = 0o750

	path := socketIn(t)
	configure(t)

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

func TestServeNeedsItsSettings(t *testing.T) {
	tests := []struct{ name, key string }{
		{"NUDGE_LOGIN", "login"},
		{"NUDGE_HOST", "host"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := socketIn(t)
			configure(t)
			t.Setenv(test.name, "")

			var stdout, stderr strings.Builder

			code := cli.Run(t.Context(), []string{"serve"}, &stdout, &stderr)

			want := fmt.Sprintf("nudge serve: %s is not set, and %s has no %q\n",
				test.name, settingsPath(t), test.key)
			if code != 1 || stderr.String() != want {
				t.Errorf("Run(serve) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
			}

			assertGone(t, path)
		})
	}
}

func TestServeNeedsAValidSettingsFile(t *testing.T) {
	path := socketIn(t)
	configure(t)

	file := settingsPath(t)
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file, []byte(`{"port": 8700}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr strings.Builder

	code := cli.Run(t.Context(), []string{"serve"}, &stdout, &stderr)
	if want := "nudge serve: the settings file " + file + " is invalid: "; code != 1 ||
		!strings.HasPrefix(stderr.String(), want) {
		t.Errorf("Run(serve) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
	}

	assertGone(t, path)
}

func TestServeNeedsItsKey(t *testing.T) {
	path := socketIn(t)
	configure(t)
	standIn(t, "locked")

	var stdout, stderr strings.Builder

	code := cli.Run(t.Context(), []string{"serve"}, &stdout, &stderr)

	want := "nudge serve: read the VAPID key from the Keychain " +
		"(service nudge, account vapid-private-key): exit status 36: " +
		"security: SecKeychainSearchCopyNext: User interaction is not allowed.\n"
	if code != 1 || stderr.String() != want {
		t.Errorf("Run(serve) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
	}

	assertGone(t, path)
}

func TestServeNeedsValidSubscriptions(t *testing.T) {
	path := socketIn(t)
	configure(t)

	file := filepath.Join(t.TempDir(), "subscriptions.json")
	if err := os.WriteFile(file, []byte("["), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("NUDGE_SUBSCRIPTIONS", file)

	var stdout, stderr strings.Builder

	code := cli.Run(t.Context(), []string{"serve"}, &stdout, &stderr)
	if want := "nudge serve: the subscriptions file " + file + " is invalid: "; code != 1 ||
		!strings.HasPrefix(stderr.String(), want) {
		t.Errorf("Run(serve) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
	}

	assertGone(t, path)
}

func TestServeNeedsThePhoneDoorsAddress(t *testing.T) {
	var config net.ListenConfig

	taken, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = taken.Close() })

	tests := []struct{ name, listen string }{
		{"a taken address", taken.Addr().String()},
		{"an address without a port", "127.0.0.1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := socketIn(t)
			configure(t)
			t.Setenv("NUDGE_LISTEN", test.listen)

			var stdout, stderr strings.Builder

			code := cli.Run(t.Context(), []string{"serve"}, &stdout, &stderr)
			if want := "nudge serve: open the phone door: "; code != 1 ||
				!strings.HasPrefix(stderr.String(), want) {
				t.Errorf("Run(serve) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
			}

			assertGone(t, path)
		})
	}
}
