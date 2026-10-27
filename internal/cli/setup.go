package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/codedogapp/nudge/internal/setup"
)

const afterSetup = `Start a Copilot session with /nudge to ask on the phone (docs/setup.md).`

// setupJob is what nudge setup does: the commands it runs, and the server it points at.
type setupJob struct {
	run    setup.Run
	server string
}

// runSetup points this machine at the nudge server, adds nudge to Copilot CLI, prints what each
// step did, and stops at the first that fails.
func (job setupJob) runSetup(ctx context.Context, stdout, stderr io.Writer) int {
	config, err := os.UserConfigDir()
	if err != nil {
		return failed(stderr, "setup", fmt.Errorf("find the config folder: %w", err))
	}

	bin, err := os.Executable()
	if err != nil {
		return failed(stderr, "setup", fmt.Errorf("find this program: %w", err))
	}

	skills := filepath.Join(config, "nudge", "skills")

	steps := []func() ([]string, error){
		func() ([]string, error) { return one(setup.WriteSettings(settingsFile(config), job.server)) },
		func() ([]string, error) { return nil, setup.WriteSkill(skills) },
		func() ([]string, error) { return setup.AddToCopilot(ctx, job.run, bin, skills) },
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
