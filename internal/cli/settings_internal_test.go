package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate gives the test a home of its own, with no settings file, and empties nudge serve's
// variables. It returns the config folder.
func isolate(t *testing.T) string {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	for _, name := range []string{loginVar, hostVar, listenVar, securityVar, subscriptionsVar} {
		t.Setenv(name, "")
	}

	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	return config
}

// writeSettings writes data to the settings file in the config folder.
func writeSettings(t *testing.T, config, data string) {
	t.Helper()

	path := settingsFile(config)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}

	writeFile(t, path, data)
}

func TestReadSettingsTakesTheVariableThenTheFileThenTheDefault(t *testing.T) {
	variables := map[string]string{
		loginVar:         "env@example.com",
		hostVar:          "env.tailnet.ts.net",
		listenVar:        "127.0.0.1:8799",
		securityVar:      "/stand-in/security",
		subscriptionsVar: "/env/subscriptions.json",
	}
	fromVariables := settings{
		login:         "env@example.com",
		host:          "env.tailnet.ts.net",
		listen:        "127.0.0.1:8799",
		security:      "/stand-in/security",
		subscriptions: "/env/subscriptions.json",
	}

	const everything = `{"login": "file@example.com", "host": "file.tailnet.ts.net",
		"listen": "127.0.0.1:8798", "subscriptions": "/file/subscriptions.json"}`

	tests := []struct {
		name string
		vars map[string]string
		file string
		want settings // An empty subscriptions path stands for the default one.
	}{
		{name: "the variables", vars: variables, want: fromVariables},
		{
			name: "the variables over the file",
			vars: variables,
			file: everything,
			want: fromVariables,
		},
		{
			name: "the file",
			file: everything,
			want: settings{
				login:         "file@example.com",
				host:          "file.tailnet.ts.net",
				listen:        "127.0.0.1:8798",
				security:      "/usr/bin/security",
				subscriptions: "/file/subscriptions.json",
			},
		},
		{
			name: "some from each",
			vars: map[string]string{loginVar: "env@example.com", listenVar: "127.0.0.1:8799"},
			file: `{"host": "file.tailnet.ts.net", "subscriptions": "/file/subscriptions.json"}`,
			want: settings{
				login:         "env@example.com",
				host:          "file.tailnet.ts.net",
				listen:        "127.0.0.1:8799",
				security:      "/usr/bin/security",
				subscriptions: "/file/subscriptions.json",
			},
		},
		{
			name: "the defaults",
			vars: map[string]string{loginVar: "env@example.com", hostVar: "env.tailnet.ts.net"},
			want: settings{
				login:    "env@example.com",
				host:     "env.tailnet.ts.net",
				listen:   "127.0.0.1:8700",
				security: "/usr/bin/security",
			},
		},
		{
			name: "the defaults over empty values in the file",
			vars: map[string]string{loginVar: "env@example.com", hostVar: "env.tailnet.ts.net"},
			file: `{"login": "", "host": "", "listen": "", "subscriptions": ""}`,
			want: settings{
				login:    "env@example.com",
				host:     "env.tailnet.ts.net",
				listen:   "127.0.0.1:8700",
				security: "/usr/bin/security",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := isolate(t)

			for name, value := range test.vars {
				t.Setenv(name, value)
			}

			if test.file != "" {
				writeSettings(t, config, test.file)
			}

			want := test.want
			if want.subscriptions == "" {
				want.subscriptions = filepath.Join(config, "nudge", "subscriptions.json")
			}

			if got, err := readSettings(); err != nil || got != want {
				t.Errorf("readSettings() = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

func TestReadSettingsNeedsALoginAndAHost(t *testing.T) {
	const (
		noLogin = `NUDGE_LOGIN is not set, and %s has no "login"`
		noHost  = `NUDGE_HOST is not set, and %s has no "host"`
	)

	tests := []struct{ name, login, file, want string }{
		{name: "nothing", want: noLogin},
		{
			name: "an empty login",
			file: `{"login": "", "host": "file.tailnet.ts.net"}`,
			want: noLogin,
		},
		{name: "no host", login: "env@example.com", want: noHost},
		{name: "an empty host", file: `{"login": "file@example.com", "host": ""}`, want: noHost},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := isolate(t)
			t.Setenv(loginVar, test.login)

			if test.file != "" {
				writeSettings(t, config, test.file)
			}

			_, err := readSettings()
			if want := fmt.Sprintf(test.want, settingsFile(config)); err == nil ||
				err.Error() != want {
				t.Errorf("readSettings() = %v, want %q", err, want)
			}
		})
	}
}

func TestReadSettingsNeedsAValidFile(t *testing.T) {
	config := isolate(t)
	t.Setenv(loginVar, "env@example.com")
	t.Setenv(hostVar, "env.tailnet.ts.net")
	writeSettings(t, config, `{"port": 8700}`)

	_, err := readSettings()
	if want := "the settings file " + settingsFile(config) + " is invalid: "; err == nil ||
		!strings.HasPrefix(err.Error(), want) {
		t.Errorf("readSettings() = %v, want an error that starts with %q", err, want)
	}
}

func TestReadSettingsNeedsAConfigFolder(t *testing.T) {
	isolate(t)
	t.Setenv(loginVar, "env@example.com")
	t.Setenv(hostVar, "env.tailnet.ts.net")
	t.Setenv(subscriptionsVar, "/env/subscriptions.json")
	t.Setenv("HOME", "")

	_, err := readSettings()
	if want := "find the config folder: "; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("readSettings() = %v, want an error that starts with %q", err, want)
	}
}
