package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes data to name, with read and write permission for the user only.
func writeFile(t *testing.T, name, data string) {
	t.Helper()

	if err := os.WriteFile(name, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadSettingsFileReadsTheSettings(t *testing.T) {
	tests := []struct {
		name string
		data string
		want fileSettings
	}{
		{
			name: "every key",
			data: `{"login": "user@example.com", "host": "machine.tailnet.ts.net",
				"listen": "127.0.0.1:8799", "subscriptions": "/stand-in/subscriptions.json",
				"server": "https://nudge.tailnet.ts.net"}`,
			want: fileSettings{
				Login:         "user@example.com",
				Host:          "machine.tailnet.ts.net",
				Listen:        "127.0.0.1:8799",
				Subscriptions: "/stand-in/subscriptions.json",
				Server:        "https://nudge.tailnet.ts.net",
			},
		},
		{
			name: "some keys",
			data: `{"host": "machine.tailnet.ts.net"}`,
			want: fileSettings{Host: "machine.tailnet.ts.net"},
		},
		{name: "no keys", data: `{}`},
		{name: "null", data: `null`},
		{name: "a null value", data: `{"login": null}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			writeFile(t, path, test.data)

			if got, err := readSettingsFile(path); err != nil || got != test.want {
				t.Errorf("readSettingsFile() = %+v, %v; want %+v", got, err, test.want)
			}
		})
	}
}

func TestReadSettingsFileWithoutAFile(t *testing.T) {
	tests := []struct{ name, path string }{
		{"no file", "settings.json"},
		{"no folder", filepath.Join("nudge", "settings.json")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.path)

			if got, err := readSettingsFile(path); err != nil || got != (fileSettings{}) {
				t.Errorf("readSettingsFile() = %+v, %v; want no settings", got, err)
			}
		})
	}
}

func TestReadSettingsFileRefusesAnInvalidFile(t *testing.T) {
	tests := []struct {
		name string
		data string
		end  string // The end of the error, when nudge words it.
	}{
		{name: "an unknown key", data: `{"login": "user@example.com", "port": 8700}`},
		{name: "a key in another case", data: `{"Login": "user@example.com"}`},
		{name: "a duplicate key", data: `{"login": "a@example.com", "login": "b@example.com"}`},
		{name: "a value that isn't a string", data: `{"listen": 8700}`},
		{name: "an array", data: `[]`},
		{name: "trailing data", data: `{} {}`},
		{name: "an empty file", data: ``},
		{name: "invalid UTF-8", data: "{\"login\": \"\xff\"}"},
		{
			name: "a relative subscriptions path",
			data: `{"subscriptions": "subscriptions.json"}`,
			end:  `subscriptions "subscriptions.json" isn't an absolute path`,
		},
		{
			name: "a subscriptions path under ~",
			data: `{"subscriptions": "~/subscriptions.json"}`,
			end:  `subscriptions "~/subscriptions.json" isn't an absolute path`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			writeFile(t, path, test.data)

			_, err := readSettingsFile(path)

			start := "the settings file " + path + " is invalid: "
			if err == nil || !strings.HasPrefix(err.Error(), start) ||
				!strings.HasSuffix(err.Error(), test.end) {
				t.Errorf(
					"readSettingsFile() = %v, want an error that starts with %q and ends with %q",
					err,
					start,
					test.end,
				)
			}
		})
	}
}

func TestReadSettingsFileRefusesAPathItCantRead(t *testing.T) {
	tests := []struct {
		name string
		make func(t *testing.T, path string)
	}{
		{"a folder", func(t *testing.T, path string) {
			t.Helper()

			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"a file without read permission", func(t *testing.T, path string) {
			t.Helper()

			if os.Geteuid() == 0 {
				t.Skip("root reads every file")
			}

			if err := os.WriteFile(path, []byte(`{}`), 0o200); err != nil {
				t.Fatal(err)
			}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			test.make(t, path)

			_, err := readSettingsFile(path)
			if err == nil || !strings.HasPrefix(err.Error(), "read the settings file: ") ||
				!strings.Contains(err.Error(), path) {
				t.Errorf("readSettingsFile() = %v, want an error that names %s", err, path)
			}
		})
	}
}
