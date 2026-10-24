// Package cli is nudge's command line: it runs the command that the arguments name, and picks the
// exit code.
package cli

import (
	"context"
	"fmt"
	"io"
)

const usage = `usage: nudge serve
       nudge ask [--option TEXT]... [--details TEXT] QUESTION`

// The exit codes: done, an error, and a usage error.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// Run runs the command that args name until it ends or ctx ends, and returns its exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return misused(stderr)
	}

	switch args[0] {
	case "serve":
		return serve(ctx, args[1:], stderr)
	case "ask":
		return ask(ctx, args[1:], stdout, stderr)
	default:
		return misused(stderr)
	}
}

// misused prints the usage, and returns the exit code of a usage error.
func misused(stderr io.Writer) int {
	_, _ = fmt.Fprintln(stderr, usage)
	return exitUsage
}

// failed prints the command's error, and returns the exit code of an error.
func failed(stderr io.Writer, command string, err error) int {
	_, _ = fmt.Fprintf(stderr, "nudge %s: %v\n", command, err)
	return exitError
}
