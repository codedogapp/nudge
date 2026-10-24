package cli

import (
	"context"
	"io"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/nudge/internal/mcptool"
)

// serveMCP runs nudge mcp: nudge's MCP server, on transport, until the agent goes away or ctx
// ends. Its questions come from the project in the current folder, as nudge ask's do.
func serveMCP(ctx context.Context, args []string, transport mcp.Transport, stderr io.Writer) int {
	if len(args) != 0 {
		return misused(stderr)
	}

	err := mcptool.Run(ctx, version(), send, transport)
	if err != nil && ctx.Err() == nil {
		return failed(stderr, "mcp", err)
	}

	return exitOK
}

// version is nudge's version, as the build recorded it.
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}

	return "(devel)"
}
