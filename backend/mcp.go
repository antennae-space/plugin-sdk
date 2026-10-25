package backend

import (
	"context"
	"encoding/json"
	"errors"
)

// MCPTool answers an MCP tool of the manifest (contributes.mcpTools, IDE 1.24): the text the
// model reads; an error goes back to it as a failed call.
type MCPTool func(ctx context.Context, sc Scope, args json.RawMessage) (string, error)

// Scope is what an MCP call works on, found by the IDE from the client's folder.
type Scope struct {
	// Project is the project of the folder (a worktree has its own), Main its main project.
	Project string `json:"project"`
	Main    string `json:"main"`
	// Ticket is the ticket of the worktree the client works in, 0 elsewhere.
	Ticket int64 `json:"ticket"`
	// URL is the address the user opens the IDE at.
	URL string `json:"url"`
}

// Question is a question of the plugin waiting for the user, as the Questions tool shows it:
// Kind permission (Options) or form (Text and Schema, a JSON Schema of the answer).
type Question struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Name    string          `json:"name,omitempty"`
	Title   string          `json:"title"`
	Text    string          `json:"text,omitempty"`
	Options []Option        `json:"options,omitempty"`
	Schema  json.RawMessage `json:"schema,omitempty"`
	Last    string          `json:"last,omitempty"`
	Time    int64           `json:"time"`
}

// Option is a choice of a permission question; Kind allow_once, allow_always, reject_once…
type Option struct {
	ID   string `json:"optionId"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// Answer is the user's answer: the option of a permission, or the action of a form (accept with
// Content, decline, cancel).
type Answer struct {
	ID      string          `json:"id"`
	Option  string          `json:"option"`
	Action  string          `json:"action"`
	Content json.RawMessage `json:"content"`
}

// Questions are the questions of a plugin: List those waiting now, Answer one answered.
type Questions struct {
	List   func(ctx context.Context) []Question
	Answer func(ctx context.Context, a Answer) error
}

// QuestionsChanged tells the IDE its questions changed (one asked, answered, withdrawn); asked
// notifies the user's devices.
func QuestionsChanged(ctx context.Context, asked bool) error {
	return Core(ctx, "questions.changed", map[string]any{"asked": asked}, nil)
}

// Note is a push notification: Kind one the devices choose from (feedback, asks, finished,
// tests), Query the parameters of the project's address it opens (?ticket=3: a link of the
// plugin), Texts by language with English required.
type Note struct {
	Kind    string               `json:"kind"`
	Query   string               `json:"query,omitempty"`
	Tag     string               `json:"tag,omitempty"`
	Urgency string               `json:"urgency,omitempty"`
	Texts   map[string]NoteTexts `json:"texts"`
}

// NoteTexts are the title and the body of a notification in one language.
type NoteTexts struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

// Notify sends a push notification to the user's devices (grant notify).
func Notify(ctx context.Context, n Note) error { return Core(ctx, "notify", n, nil) }

// extra answers the requests of the MCP tools and the questions.
func (c *conn) extra(ctx context.Context, m message) (any, error) {
	if c.initErr != nil {
		return nil, c.initErr
	}
	switch m.Method {
	case "mcp":
		var p struct {
			Name  string          `json:"name"`
			Args  json.RawMessage `json:"args"`
			Lang  string          `json:"lang"`
			Scope Scope           `json:"scope"`
		}
		if err := json.Unmarshal(m.Params, &p); err != nil {
			return nil, err
		}
		f := c.p.MCP[p.Name]
		if f == nil {
			return Result{Content: "unknown tool: " + p.Name, Error: true}, nil
		}
		text, err := f(context.WithValue(ctx, langKey{}, p.Lang), p.Scope, p.Args)
		if err != nil {
			return Result{Content: err.Error(), Error: true}, nil
		}
		return Result{Content: text}, nil
	case "questions.list":
		if c.p.Questions == nil || c.p.Questions.List == nil {
			return []Question{}, nil
		}
		list := c.p.Questions.List(ctx)
		if list == nil {
			list = []Question{}
		}
		return list, nil
	default:
		var a Answer
		if err := json.Unmarshal(m.Params, &a); err != nil {
			return nil, err
		}
		if c.p.Questions == nil || c.p.Questions.Answer == nil {
			return nil, errors.New("this plugin asks no questions")
		}
		return map[string]any{}, c.p.Questions.Answer(ctx, a)
	}
}
