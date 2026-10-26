package mcptool_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/nudge/internal/mcptool"
)

// connectReport runs nudge's MCP server with report until the test ends, and returns a client's
// session with it.
func connectReport(t *testing.T, report mcptool.Report) *mcp.ClientSession {
	t.Helper()

	serverEnd, clientEnd := mcp.NewInMemoryTransports()

	go func() { _ = mcptool.Run(t.Context(), "v1", mcptool.Door{Report: report}, serverEnd) }()

	return join(t, clientEnd)
}

func callReport(t *testing.T, session *mcp.ClientSession, arguments any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "report", Arguments: arguments,
	})
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func TestReportSetsTheNote(t *testing.T) {
	noted := make(chan string, 1)
	session := connectReport(t, func(_ context.Context, note string) error {
		noted <- note
		return nil
	})

	result := callReport(t, session, map[string]any{"note": "tests are green"})

	if got := <-noted; got != "tests are green" || result.IsError {
		t.Errorf("note = %q, error = %v; want the note, no error", got, result.IsError)
	}
}

func TestReportErrorIsAToolError(t *testing.T) {
	session := connectReport(t, func(context.Context, string) error {
		return errors.New("reach nudge serve: no socket")
	})

	result := callReport(t, session, map[string]any{"note": "tests are green"})

	if text := textOf(t, result); !result.IsError || text != "reach nudge serve: no socket" {
		t.Errorf("result = %v %q, want the error as a tool error", result.IsError, text)
	}
}

func TestReportInputSchema(t *testing.T) {
	listed := tool(t, "report")
	input := schemaOf(t, listed.InputSchema)

	if !reflect.DeepEqual(input.Required, []string{"note"}) || len(input.Properties) != 1 {
		t.Errorf("schema = %+v, want a required note alone", input)
	}

	if got := input.Properties["note"]; !strings.Contains(string(got.Type), "string") ||
		got.Description == "" {
		t.Errorf("note = %s %q, want a described string", got.Type, got.Description)
	}

	if !strings.Contains(listed.Description, "phone") {
		t.Errorf("description = %q, want it to say who reads the note", listed.Description)
	}
}
