package mcptool_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// schema is the part of a JSON schema that the tests check.
type schema struct {
	Required   []string            `json:"required"`
	Properties map[string]property `json:"properties"`
}

// property is a property of a schema.
type property struct {
	Type        jsontext.Value `json:"type"`
	Description string         `json:"description"`
	Items       *property      `json:"items"`
}

// schemaOf reads a tool's schema as the client got it.
func schemaOf(t *testing.T, raw any) schema {
	t.Helper()

	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	var got schema
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	return got
}

// askTool lists the server's tools, and returns the one tool, ask.
func askTool(t *testing.T) *mcp.Tool {
	t.Helper()

	listed, err := connect(t, nil).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(listed.Tools) != 1 || listed.Tools[0].Name != "ask" {
		t.Fatalf("tools = %+v, want ask alone", listed.Tools)
	}

	return listed.Tools[0]
}

func TestListsTheAskTool(t *testing.T) {
	if tool := askTool(t); !strings.Contains(tool.Description, "phone") {
		t.Errorf(
			"description = %q, want it to say the question goes to the phone",
			tool.Description,
		)
	}
}

func TestAskInputSchema(t *testing.T) {
	input := schemaOf(t, askTool(t).InputSchema)

	if !reflect.DeepEqual(input.Required, []string{"question"}) || len(input.Properties) != 3 {
		t.Errorf("schema = %+v, want question, options and details, question required", input)
	}

	wantTypes := map[string]string{"question": "string", "options": "array", "details": "string"}
	for name, want := range wantTypes {
		got := input.Properties[name]
		if !strings.Contains(string(got.Type), want) || got.Description == "" {
			t.Errorf("%s = %s %q, want a described %s", name, got.Type, got.Description, want)
		}
	}

	if items := input.Properties["options"].Items; items == nil ||
		string(items.Type) != `"string"` {
		t.Errorf("options' items = %+v, want strings", items)
	}
}

func TestAskOutputSchema(t *testing.T) {
	output := schemaOf(t, askTool(t).OutputSchema)

	for _, name := range []string{"option", "text"} {
		if string(output.Properties[name].Type) != `"string"` {
			t.Errorf("%s = %s, want a string", name, output.Properties[name].Type)
		}
	}

	if len(output.Properties) != 2 {
		t.Errorf("output = %+v, want option and text", output.Properties)
	}
}
