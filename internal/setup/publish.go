package setup

import (
	"context"
	"fmt"
)

// Publish has Tailscale serve the phone door, nudge serve's default address, on port 8443 of the
// tailnet name, and returns a line that says so. Tailscale takes the same command a second time.
func Publish(ctx context.Context, run Run) (string, error) {
	if _, err := run(ctx, "tailscale", "serve", "--bg", "--https=8443", "8700"); err != nil {
		return "", fmt.Errorf("publish the phone door: %w", err)
	}

	return "tailscale: serving the phone door on port 8443", nil
}
