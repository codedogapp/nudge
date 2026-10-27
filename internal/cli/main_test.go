package cli_test

import (
	"os"
	"testing"
)

// TestMain gives the tests an empty home, so that the user's own settings file never changes
// what they do.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "nudge-cli-test")
	if err != nil {
		panic(err)
	}

	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CONFIG_HOME", home)

	code := m.Run()

	_ = os.RemoveAll(home)

	os.Exit(code)
}
