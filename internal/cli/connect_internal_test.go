package cli

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestConnectTakesTheServerFromTheVariableThenTheFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("NUDGE_SOCKET", filepath.Join(home, "agent.sock"))
	t.Setenv("NUDGE_SERVER", "")

	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(settingsFile(config)), 0o700); err != nil {
		t.Fatal(err)
	}

	viaSocket := func() bool {
		client, err := connect()
		if err != nil {
			t.Fatal(err)
		}

		_, socket := client.Transport.(*http.Transport)

		return socket
	}

	if !viaSocket() {
		t.Error("with no server set, connect didn't use the socket")
	}

	writeFile(t, settingsFile(config), `{"server": "https://nudge.tailnet.ts.net"}`)

	if viaSocket() {
		t.Error("with the file's server, connect used the socket")
	}

	t.Setenv("NUDGE_SERVER", "http://insecure")

	if _, err := connect(); err == nil {
		t.Error("the variable should win over the file, and an http URL is refused")
	}

	t.Setenv("NUDGE_SERVER", "")
	writeFile(t, settingsFile(config), `{"server": 3}`)

	if _, err := connect(); err == nil {
		t.Error("an invalid settings file gave no error")
	}
}
