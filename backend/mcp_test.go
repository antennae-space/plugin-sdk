package backend

import (
	"context"
	"encoding/json"
	"testing"
)

// An MCP tool with its scope, the questions listed and answered.
func TestMCPAndQuestions(t *testing.T) {
	var answered Answer
	p := Plugin{
		MCP: map[string]MCPTool{"greet_where": func(ctx context.Context, sc Scope, _ json.RawMessage) (string, error) {
			return sc.Project + " " + sc.Main + " " + LangOf(ctx), nil
		}},
		Questions: &Questions{
			List:   func(context.Context) []Question { return []Question{{ID: "q1", Kind: "permission", Title: "Greet?"}} },
			Answer: func(_ context.Context, a Answer) error { answered = a; return nil },
		},
	}
	f := start(t, p, func(string, json.RawMessage) (any, *Error) { return nil, nil })
	f.ask("initialize", Info{APIVersion: 1, Plugin: "greet"})
	r := f.ask("mcp", map[string]any{"name": "greet_where", "lang": "fr", "scope": Scope{Project: "p-t1", Main: "p"}})
	if string(r.Result) != `{"content":"p-t1 p fr"}` {
		t.Fatalf("mcp: %s", r.Result)
	}
	if r := f.ask("mcp", map[string]any{"name": "nope"}); string(r.Result) != `{"content":"unknown tool: nope","error":true}` {
		t.Fatalf("unknown: %s", r.Result)
	}
	if r := f.ask("questions.list", map[string]any{}); string(r.Result) != `[{"id":"q1","kind":"permission","title":"Greet?","time":0}]` {
		t.Fatalf("list: %s", r.Result)
	}
	f.ask("questions.answer", map[string]any{"id": "q1", "option": "yes"})
	if answered.ID != "q1" || answered.Option != "yes" {
		t.Fatalf("answer: %+v", answered)
	}
}
