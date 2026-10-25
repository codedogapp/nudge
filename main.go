// Command nudge hands a coding agent's questions to the user, wherever they are, and gives the
// answers back to the agent.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/codedogapp/nudge/internal/cli"
)

// version is nudge's version. Homebrew's formula sets it with -ldflags "-X main.version=v1.2.3";
// other builds leave it empty, and nudge takes the version that Go's build info recorded.
var version string

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cli.Program{Version: version}.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)

	stop()
	os.Exit(code)
}
