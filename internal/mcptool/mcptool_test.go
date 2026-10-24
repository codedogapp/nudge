package mcptool_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/mcptool"
)

// serve runs nudge's MCP server with ask on transport until ctx ends, and returns where Run's
// error arrives.
func serve(ctx context.Context, ask mcptool.Ask, transport mcp.Transport) <-chan error {
	ran := make(chan error, 1)

	go func() { ran <- mcptool.Run(ctx, "v1.2.3", ask, transport) }()

	return ran
}

// join connects a client to the server at the other end of transport, and returns its session.
func join(t *testing.T, transport mcp.Transport) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	session, err := client.Connect(t.Context(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

// connect runs nudge's MCP server with ask until the test ends, and returns a client's session
// with it.
func connect(t *testing.T, ask mcptool.Ask) *mcp.ClientSession {
	t.Helper()

	serverEnd, clientEnd := mcp.NewInMemoryTransports()
	serve(t.Context(), ask, serverEnd)

	return join(t, clientEnd)
}

// answering returns an Ask that hands each question over, and returns answer and err.
func answering(answer agentdoor.Answer, err error) (mcptool.Ask, <-chan agentdoor.Question) {
	asked := make(chan agentdoor.Question, 1)

	return func(_ context.Context, question agentdoor.Question) (agentdoor.Answer, error) {
		asked <- question
		return answer, err
	}, asked
}

// callAsk calls the ask tool with arguments.
func callAsk(t *testing.T, session *mcp.ClientSession, arguments any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "ask", Arguments: arguments,
	})
	if err != nil {
		t.Fatal(err)
	}

	return result
}

// textOf returns the text of result's one content.
func textOf(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if len(result.Content) != 1 {
		t.Fatalf("content = %v, want one text", result.Content)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T, want text", result.Content[0])
	}

	return text.Text
}

func TestServerInfo(t *testing.T) {
	info := connect(t, nil).InitializeResult().ServerInfo

	if info.Name != "nudge" || info.Version != "v1.2.3" {
		t.Errorf("server = %s %s, want nudge v1.2.3", info.Name, info.Version)
	}
}

func TestAskReturnsTheAnswer(t *testing.T) {
	answer := agentdoor.Answer{Option: "Yes", Text: "and add a test"}
	ask, asked := answering(answer, nil)

	result := callAsk(t, connect(t, ask), map[string]any{
		"question": "Merge the pull request?",
		"options":  []string{"Yes", "No"},
		"details":  "CI is green.",
	})

	want := agentdoor.Question{
		Text: "Merge the pull request?", Options: []string{"Yes", "No"}, Details: "CI is green.",
	}
	if got := <-asked; !reflect.DeepEqual(got, want) {
		t.Errorf("asked %+v, want %+v", got, want)
	}

	structured := map[string]any{"option": "Yes", "text": "and add a test"}
	if result.IsError || !reflect.DeepEqual(result.StructuredContent, structured) {
		t.Errorf("result = %v %v, want %v", result.IsError, result.StructuredContent, structured)
	}

	var got agentdoor.Answer
	if err := json.Unmarshal([]byte(textOf(t, result)), &got); err != nil || got != answer {
		t.Errorf("text = %q (%v), want the answer as JSON", textOf(t, result), err)
	}
}

func TestAskWithTheQuestionAlone(t *testing.T) {
	ask, asked := answering(agentdoor.Answer{Text: "Not yet."}, nil)

	result := callAsk(t, connect(t, ask), map[string]any{"question": "Ship it?"})

	if got := <-asked; !reflect.DeepEqual(got, agentdoor.Question{Text: "Ship it?"}) {
		t.Errorf("asked %+v, want the question alone", got)
	}

	if result.IsError {
		t.Errorf("result = %q, want the answer", textOf(t, result))
	}
}

func TestAskErrorIsAToolError(t *testing.T) {
	ask, _ := answering(agentdoor.Answer{}, errors.New("an option is empty or repeated"))

	result := callAsk(t, connect(t, ask), map[string]any{"question": "Ship it?"})

	if text := textOf(t, result); !result.IsError || text != "an option is empty or repeated" {
		t.Errorf("result = %v %q, want the error as a tool error", result.IsError, text)
	}
}

func TestAskRefusesInputThatDoesNotFit(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
	}{
		{"no question", map[string]any{}},
		{"a number for the question", map[string]any{"question": 3}},
		{"options that aren't a list", map[string]any{"question": "Ship it?", "options": "Yes"}},
		{
			"options that aren't strings",
			map[string]any{"question": "Ship it?", "options": []int{1}},
		},
		{"a project", map[string]any{"question": "Ship it?", "project": "atlas"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ask, asked := answering(agentdoor.Answer{Text: "Yes"}, nil)

			result := callAsk(t, connect(t, ask), test.arguments)

			if !result.IsError || len(asked) != 0 {
				t.Errorf("result = %v %q, asked %d; want a tool error, nothing asked",
					result.IsError, textOf(t, result), len(asked))
			}
		})
	}
}
