package mcptool

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// reportDescription tells the model what the report tool is for.
const reportDescription = "Tell the user how your work is going, in a line or two, " +
	"for them to read on their phone when they check on you. " +
	"Call it when you start a task and at each milestone. " +
	"It returns at once: it asks nothing and gets no answer."

// reportInput is a call of the report tool, as the agent sends it.
type reportInput struct {
	Note string `json:"note" jsonschema:"how the work is going"`
}

// handle runs a call of the report tool: it sets the note.
func (report Report) handle(
	ctx context.Context, _ *mcp.CallToolRequest, given reportInput,
) (*mcp.CallToolResult, any, error) {
	if err := report(ctx, given.Note); err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "reported"}}}, nil, nil
}
