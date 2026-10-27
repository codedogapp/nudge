package cli

import (
	"context"
	"errors"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/mcptool"
)

// reporting returns the report tool's function. It starts a session for the project in the
// current folder, which lasts until ctx ends. When it can't tell where to start one, each report
// fails with why.
func reporting(ctx context.Context) mcptool.Report {
	project, projectErr := currentProject()
	client, clientErr := connect()

	if err := errors.Join(projectErr, clientErr); err != nil {
		return func(context.Context, string) error { return err }
	}

	reporter := agentdoor.NewReporter(ctx, client, project)

	// nudge serve may not be running yet; the first report starts the session then.
	_ = reporter.Register()

	return reporter.Report
}
