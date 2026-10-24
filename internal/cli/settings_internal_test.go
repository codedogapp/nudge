package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSettingsFillsTheDefaults(t *testing.T) {
	t.Setenv(loginVar, "user@example.com")
	t.Setenv(hostVar, "machine.tailnet.ts.net")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		listen        string
		security      string
		subscriptions string
		want          settings
	}{
		{
			name: "unset",
			want: settings{
				listen:        "127.0.0.1:8700",
				security:      "/usr/bin/security",
				subscriptions: filepath.Join(config, "nudge", "subscriptions.json"),
			},
		},
		{
			name:          "set",
			listen:        "127.0.0.1:8799",
			security:      "/stand-in/security",
			subscriptions: "/stand-in/subscriptions.json",
			want: settings{
				listen:        "127.0.0.1:8799",
				security:      "/stand-in/security",
				subscriptions: "/stand-in/subscriptions.json",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(listenVar, test.listen)
			t.Setenv(securityVar, test.security)
			t.Setenv(subscriptionsVar, test.subscriptions)

			want := test.want
			want.login, want.host = "user@example.com", "machine.tailnet.ts.net"

			if got, err := readSettings(); err != nil || got != want {
				t.Errorf("readSettings() = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

func TestReadSettingsNeedsAConfigFolderForTheSubscriptions(t *testing.T) {
	t.Setenv(loginVar, "user@example.com")
	t.Setenv(hostVar, "machine.tailnet.ts.net")
	t.Setenv(subscriptionsVar, "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	_, err := readSettings()
	if want := "find the config folder: "; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("readSettings() = %v, want an error that starts with %q", err, want)
	}
}
