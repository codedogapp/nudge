package cli

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codedogapp/nudge/internal/vapid"
)

// The variables that nudge serve reads its settings from (docs/setup.md), and the phone door's
// address when neither NUDGE_LISTEN nor the settings file gives one.
const (
	loginVar         = "NUDGE_LOGIN"
	hostVar          = "NUDGE_HOST"
	listenVar        = "NUDGE_LISTEN"
	securityVar      = "NUDGE_SECURITY"
	subscriptionsVar = "NUDGE_SUBSCRIPTIONS"
	keyFileVar       = "NUDGE_VAPID_KEY_FILE"
	defaultListen    = "127.0.0.1:8700"
)

// settings are what nudge serve needs to open the phone door, to load its VAPID key and to keep
// the devices that subscribe to its pushes.
type settings struct {
	login         string // The one Tailscale login let in.
	host          string // The ts.net name that tailscale serve publishes.
	listen        string // The phone door's address.
	security      string // The security tool that keeps the VAPID key.
	subscriptions string // The file that keeps the push subscriptions.
	keyFile       string // The file that keeps the VAPID key; empty for the Keychain.
}

// readSettings reads each setting from its variable, else from the settings file, else takes its
// default. An empty value counts as unset. The login and the host have no default.
func readSettings() (settings, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return settings{}, fmt.Errorf("find the config folder: %w", err)
	}

	path := settingsFile(config)

	file, err := readSettingsFile(path)
	if err != nil {
		return settings{}, err
	}

	given := settings{
		login:    cmp.Or(os.Getenv(loginVar), file.Login),
		host:     cmp.Or(os.Getenv(hostVar), file.Host),
		listen:   cmp.Or(os.Getenv(listenVar), file.Listen, defaultListen),
		security: cmp.Or(os.Getenv(securityVar), vapid.Security),
		keyFile:  os.Getenv(keyFileVar),
		subscriptions: cmp.Or(os.Getenv(subscriptionsVar), file.Subscriptions,
			filepath.Join(config, "nudge", "subscriptions.json")),
	}

	if given.login == "" {
		return settings{}, missing(loginVar, path, "login")
	}

	if given.host == "" {
		return settings{}, missing(hostVar, path, "host")
	}

	return given, nil
}

// missing is the error for a setting that neither its variable nor the settings file gives.
func missing(name, path, key string) error {
	return fmt.Errorf("%s is not set, and %s has no %q", name, path, key)
}
