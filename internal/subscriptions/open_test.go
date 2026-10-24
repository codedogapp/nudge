package subscriptions_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/subscriptions"
)

func TestOpenRefusesAnInvalidFile(t *testing.T) {
	tests := []struct{ name, data string }{
		{"an empty file", ""},
		{"not JSON", "["},
		{"not a list", `{"endpoint": "` + first + `"}`},
		{
			"an invalid subscription",
			`[{"endpoint": "http://push.example/send/1", ` +
				`"keys": {"p256dh": "` + p256dh + `", "auth": "` + auth + `"}}]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := pathIn(t)
			if err := folder(
				t,
				path,
			).WriteFile(filepath.Base(path), []byte(test.data), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := subscriptions.Open(path)
			if want := "the subscriptions file " + path + " is invalid: "; err == nil ||
				!strings.HasPrefix(err.Error(), want) {
				t.Errorf("Open() = %v, want an error that starts with %q", err, want)
			}
		})
	}
}

func TestOpenReportsWhatFailed(t *testing.T) {
	notAFolder := pathIn(t)
	if err := folder(t, notAFolder).WriteFile(filepath.Base(notAFolder), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	aFolder := t.TempDir()

	tests := []struct{ name, path, want string }{
		{
			"a file in the folder's place",
			filepath.Join(notAFolder, "subscriptions.json"),
			"make the subscriptions folder: ",
		},
		{"a folder in the file's place", aFolder, "read the subscriptions file " + aFolder + ": "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := subscriptions.Open(test.path); err == nil ||
				!strings.HasPrefix(err.Error(), test.want) {
				t.Errorf("Open() = %v, want an error that starts with %q", err, test.want)
			}
		})
	}
}
