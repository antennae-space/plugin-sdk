package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// pod is a fake IDE speaking the protocol of web-ide's pod/plugins/backend: it asks with
// integer ids, answers `core` and collects `emit`.
type pod struct {
	t      *testing.T
	w      io.Writer
	lines  chan message
	core   func(method string, params json.RawMessage) (any, *Error)
	events chan message
	next   int
	done   chan error
}

func start(t *testing.T, p Plugin, core func(string, json.RawMessage) (any, *Error)) *pod {
	t.Helper()
	toB, fromPod := io.Pipe()
	fromB, toPod := io.Pipe()
	f := &pod{t: t, w: fromPod, lines: make(chan message, 16), core: core, events: make(chan message, 16), done: make(chan error, 1)}
	go func() { f.done <- ServeIO(context.Background(), p, toB, toPod); toPod.Close() }()
	go func() {
		sc := bufio.NewScanner(fromB)
		for sc.Scan() {
			var m message
			if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
				t.Errorf("not JSON: %s", sc.Bytes())
				continue
			}
			if m.JSONRPC != "2.0" {
				t.Errorf("no jsonrpc 2.0: %s", sc.Bytes())
			}
			switch m.Method {
			case "core":
				var c struct {
					Method string          `json:"method"`
					Params json.RawMessage `json:"params"`
				}
				_ = json.Unmarshal(m.Params, &c)
				res, e := f.core(c.Method, c.Params)
				out := message{JSONRPC: "2.0", ID: m.ID, Error: e}
				if e == nil {
					out.Result, _ = json.Marshal(res)
				}
				f.write(out)
			case "emit":
				f.events <- m
			default:
				f.lines <- m
			}
		}
	}()
	t.Cleanup(func() { fromPod.Close() })
	return f
}

func (f *pod) write(m message) {
	data, _ := json.Marshal(m)
	if _, err := f.w.Write(append(data, '\n')); err != nil {
		f.t.Errorf("write: %v", err)
	}
}

// ask sends a request and waits for its answer.
func (f *pod) ask(method string, params any) message {
	f.t.Helper()
	f.next++
	id, _ := json.Marshal(f.next)
	raw, _ := json.Marshal(params)
	f.write(message{JSONRPC: "2.0", ID: id, Method: method, Params: raw})
	select {
	case m := <-f.lines:
		if string(m.ID) != string(id) {
			f.t.Fatalf("answer %s for the request %s", m.ID, id)
		}
		return m
	case <-time.After(5 * time.Second):
		f.t.Fatalf("no answer to %s", method)
	}
	return message{}
}

var initParams = map[string]any{"apiVersion": 1, "plugin": "greet", "version": "1.0.0", "grants": []string{"core:fs.list"},
	"project": map[string]any{"id": "p1", "root": "/home/me/proj", "local": true}}

func TestServe(t *testing.T) {
	var got Info
	p := Plugin{
		Init: func(ctx context.Context, info Info) error { got = info; return nil },
		Methods: map[string]Method{
			"greet.count": func(ctx context.Context, params json.RawMessage) (any, error) {
				var entries []string
				if err := Core(ctx, "fs.list", map[string]string{"path": ""}, &entries); err != nil {
					return nil, err
				}
				if err := Emit(ctx, "greet.counted", len(entries)); err != nil {
					return nil, err
				}
				return map[string]any{"n": len(entries), "lang": LangOf(ctx), "granted": InfoOf(ctx).Granted("core:fs.list")}, nil
			},
			"greet.fail": func(ctx context.Context, params json.RawMessage) (any, error) {
				return nil, &Error{Code: 42, Message: "coded"}
			},
			"greet.denied": func(ctx context.Context, params json.RawMessage) (any, error) {
				return nil, Core(ctx, "fs.write", nil, nil)
			},
		},
		Tools: map[string]Tool{
			"greet_count": func(ctx context.Context, args json.RawMessage) (Result, error) {
				return Result{Content: "3 entries " + string(args), Summary: "3 entries"}, nil
			},
			"greet_broken": func(ctx context.Context, args json.RawMessage) (Result, error) {
				return Result{}, errors.New("broken")
			},
		},
	}
	f := start(t, p, func(method string, params json.RawMessage) (any, *Error) {
		if method != "fs.list" {
			return nil, &Error{Code: -32000, Message: method + " is not granted"}
		}
		return []string{"a", "b", "c"}, nil
	})

	if m := f.ask("initialize", initParams); m.Error != nil {
		t.Fatalf("initialize: %v", m.Error)
	}
	if got.Plugin != "greet" || got.Project.Root != "/home/me/proj" || !got.Project.Local || len(got.Grants) != 1 {
		t.Fatalf("info: %+v", got)
	}

	m := f.ask("call", map[string]any{"method": "greet.count", "params": nil, "lang": "fr"})
	if m.Error != nil || string(m.Result) != `{"granted":true,"lang":"fr","n":3}` {
		t.Fatalf("call: %s %v", m.Result, m.Error)
	}
	select {
	case e := <-f.events:
		if string(e.Params) != `{"data":3,"event":"greet.counted"}` || e.ID != nil {
			t.Fatalf("emit: %s", e.Params)
		}
	case <-time.After(time.Second):
		t.Fatal("no emit")
	}

	if m := f.ask("call", map[string]any{"method": "greet.fail"}); m.Error == nil || m.Error.Code != 42 || m.Error.Message != "coded" {
		t.Fatalf("coded error: %+v", m.Error)
	}
	if m := f.ask("call", map[string]any{"method": "greet.denied"}); m.Error == nil || !strings.Contains(m.Error.Message, "not granted") {
		t.Fatalf("core refused: %+v", m.Error)
	}
	if m := f.ask("call", map[string]any{"method": "greet.nope"}); m.Error == nil || m.Error.Code != -32601 {
		t.Fatalf("unknown method: %+v", m.Error)
	}

	m = f.ask("tool", map[string]any{"name": "greet_count", "args": map[string]any{}, "lang": ""})
	if string(m.Result) != `{"content":"3 entries {}","summary":"3 entries"}` {
		t.Fatalf("tool: %s", m.Result)
	}
	m = f.ask("tool", map[string]any{"name": "greet_broken"})
	if string(m.Result) != `{"content":"broken","summary":"failed","error":true}` {
		t.Fatalf("tool error: %s", m.Result)
	}
	m = f.ask("tool", map[string]any{"name": "greet_nope"})
	var r Result
	_ = json.Unmarshal(m.Result, &r)
	if !r.Error {
		t.Fatalf("unknown tool: %s", m.Result)
	}

	if m := f.ask("shutdown", nil); m.Error != nil {
		t.Fatalf("shutdown: %v", m.Error)
	}
	select {
	case err := <-f.done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeIO did not return at shutdown")
	}
}

// Init may call the core; a failing Init fails the calls after it.
func TestInit(t *testing.T) {
	p := Plugin{
		Init: func(ctx context.Context, info Info) error {
			var root []string
			if err := Core(ctx, "fs.list", nil, &root); err != nil {
				return err
			}
			return errors.New("no config in " + strings.Join(root, ","))
		},
		Methods: map[string]Method{"greet.x": func(context.Context, json.RawMessage) (any, error) { return 1, nil }},
	}
	f := start(t, p, func(string, json.RawMessage) (any, *Error) { return []string{"a"}, nil })
	if m := f.ask("initialize", initParams); m.Error == nil || m.Error.Message != "no config in a" {
		t.Fatalf("initialize: %+v", m.Error)
	}
	if m := f.ask("call", map[string]any{"method": "greet.x"}); m.Error == nil || m.Error.Message != "no config in a" {
		t.Fatalf("call after a failed Init: %+v", m.Error)
	}
}

// A call sent before initialize waits for it.
func TestCallWaitsForInit(t *testing.T) {
	release := make(chan struct{})
	p := Plugin{
		Init:    func(context.Context, Info) error { <-release; return nil },
		Methods: map[string]Method{"greet.x": func(ctx context.Context, _ json.RawMessage) (any, error) { return InfoOf(ctx).Plugin, nil }},
	}
	f := start(t, p, nil)
	f.write(message{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize", Params: mustJSON(initParams)})
	f.write(message{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "call", Params: mustJSON(map[string]any{"method": "greet.x"})})
	select {
	case m := <-f.lines:
		t.Fatalf("answered before Init: %s", m.ID)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	seen := map[string]string{}
	for range 2 {
		m := <-f.lines
		seen[string(m.ID)] = string(m.Result)
	}
	if seen["2"] != `"greet"` {
		t.Fatalf("call: %v", seen)
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
