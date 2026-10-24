// Package mcptool is nudge's MCP server: it offers agents one tool, ask, which asks the user a
// question and returns the answer.
package mcptool

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/nudge/internal/agentdoor"
)

// Ask asks question, and waits for its answer until ctx ends.
type Ask func(ctx context.Context, question agentdoor.Question) (agentdoor.Answer, error)

// description tells the model what the ask tool is for.
const description = "Ask the user a question on their phone, and wait for the answer. " +
	"Use it for a decision or context that only the user can give. " +
	"Offer the likely answers as options: the user can pick one, type a reply, or both. " +
	"The call waits until the user answers, which can take minutes or hours."

// input is a call of the ask tool, as the agent sends it.
type input struct {
	Question string   `json:"question"          jsonschema:"the question to ask the user"`
	Options  []string `json:"options,omitempty" jsonschema:"answers the user can pick with one tap"`
	Details  string   `json:"details,omitempty" jsonschema:"what the user needs to know to answer"`
}

// Run serves nudge's MCP server at version on transport, until the agent goes away or ctx ends.
// Its ask tool asks through ask.
func Run(ctx context.Context, version string, ask Ask, transport mcp.Transport) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "nudge", Version: version}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "ask", Description: description}, ask.until(ctx).handle)

	return server.Run(ctx, transport)
}

// until returns ask, with each question ending when stop ends too. The SDK doesn't end a call
// when the server's context ends, and the server waits for its calls before it stops.
func (ask Ask) until(stop context.Context) Ask {
	return func(ctx context.Context, question agentdoor.Question) (agentdoor.Answer, error) {
		call, cancel := context.WithCancel(ctx)
		defer cancel()
		defer context.AfterFunc(stop, cancel)()

		return ask(call, question)
	}
}

// handle runs a call of the ask tool: it asks the call's question, and returns the answer. When
// the agent cancels the call or goes away, ctx ends, and so does the question.
func (ask Ask) handle(
	ctx context.Context, _ *mcp.CallToolRequest, given input,
) (*mcp.CallToolResult, agentdoor.Answer, error) {
	answer, err := ask(ctx, agentdoor.Question{
		Text:    given.Question,
		Options: given.Options,
		Details: given.Details,
	})

	return nil, answer, err
}
