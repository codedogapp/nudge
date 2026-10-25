package cli

import (
	"cmp"
	"context"
	"io"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/nudge/internal/mcptool"
)

// serveMCP runs nudge mcp: nudge's MCP server, on transport, until the agent goes away or ctx
// ends. Its questions come from the project in the current folder, as nudge ask's do.
func (p Program) serveMCP(
	ctx context.Context,
	args []string,
	transport mcp.Transport,
	stderr io.Writer,
) int {
	if len(args) != 0 {
		return misused(stderr)
	}

	err := mcptool.Run(ctx, p.version(), send, transport)
	if err != nil && ctx.Err() == nil {
		return failed(stderr, "mcp", err)
	}

	return exitOK
}

// version is the version that nudge mcp reports: the one that the build set, else the one that
// Go's build info recorded.
func (p Program) version() string {
	var recorded string
	if info, ok := debug.ReadBuildInfo(); ok {
		recorded = info.Main.Version
	}

	return cmp.Or(p.Version, recorded, "(devel)")
}
