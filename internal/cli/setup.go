package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/codedogapp/nudge/internal/setup"
)

const afterSetup = `Next: restart the service if it runs (brew services restart codedogapp/tap/nudge),
open the page on the iPhone, add it to the Home Screen and turn notifications on (docs/setup.md).`

// runSetup does the setup steps in order, prints what each did, and stops at the first that fails.
func runSetup(ctx context.Context, run setup.Run, stdout, stderr io.Writer) int {
	config, err := os.UserConfigDir()
	if err != nil {
		return failed(stderr, "setup", fmt.Errorf("find the config folder: %w", err))
	}

	steps := []func() ([]string, error){
		func() ([]string, error) { return one(setup.WriteSettings(ctx, run, settingsFile(config))) },
		func() ([]string, error) { return one(setup.Publish(ctx, run)) },
		func() ([]string, error) { return setup.AddToCopilot(ctx, run) },
	}

	for _, step := range steps {
		lines, err := step()
		for _, line := range lines {
			_, _ = fmt.Fprintln(stdout, line)
		}

		if err != nil {
			return failed(stderr, "setup", err)
		}
	}

	_, _ = fmt.Fprintln(stdout, afterSetup)

	return exitOK
}

// one wraps a step's single line as its lines.
func one(line string, err error) ([]string, error) {
	return []string{line}, err
}
