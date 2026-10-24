package cli

import (
	"cmp"
	"fmt"
	"os"

	"github.com/codedogapp/nudge/internal/vapid"
)

// The variables that nudge serve reads its settings from (docs/setup.md), and the phone door's
// address when NUDGE_LISTEN is unset.
const (
	loginVar      = "NUDGE_LOGIN"
	hostVar       = "NUDGE_HOST"
	listenVar     = "NUDGE_LISTEN"
	securityVar   = "NUDGE_SECURITY"
	defaultListen = "127.0.0.1:8700"
)

// settings are what nudge serve needs to open the phone door and to load its VAPID key.
type settings struct {
	login    string // The one Tailscale login let in.
	host     string // The ts.net name that tailscale serve publishes.
	listen   string // The phone door's address.
	security string // The security tool that keeps the VAPID key.
}

// readSettings reads the settings from the environment. NUDGE_LOGIN and NUDGE_HOST must be set.
func readSettings() (settings, error) {
	login, err := required(loginVar)
	if err != nil {
		return settings{}, err
	}

	host, err := required(hostVar)
	if err != nil {
		return settings{}, err
	}

	return settings{
		login:    login,
		host:     host,
		listen:   cmp.Or(os.Getenv(listenVar), defaultListen),
		security: cmp.Or(os.Getenv(securityVar), vapid.Security),
	}, nil
}

// required returns the value of the variable name, or an error if it is unset or empty.
func required(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s is not set", name)
	}

	return value, nil
}
