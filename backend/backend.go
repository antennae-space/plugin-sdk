// Package backend runs the backend of an external plugin of Web IDE: the program the IDE starts
// confined (Landlock and seccomp) and speaks to in JSON-RPC 2.0 over stdin and stdout, one
// message per line.
//
// The IDE asks it `initialize`, `call` (an `<id>.*` method of one of its views), `tool` (one of
// its assistant tools) and `shutdown`; the backend asks the IDE `core` (a method of the core,
// refused without the grant `core:<method>`) and tells it `emit` (an `<id>.*` event to the
// windows of the project). Serve does all of it: a plugin gives its methods and tools.
//
//	func main() {
//		backend.Serve(backend.Plugin{
//			Methods: map[string]backend.Method{"greet.count": count},
//			Tools:   map[string]backend.Tool{"greet_count": countTool},
//		})
//	}
package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

// APIVersion is the version of the plugin API this package speaks.
const APIVersion = 1

// MaxMessage caps one message of either side, as the IDE does.
const MaxMessage = 8 << 20

// Info is what the IDE tells the backend at `initialize`.
type Info struct {
	APIVersion int      `json:"apiVersion"`
	Plugin     string   `json:"plugin"`
	Version    string   `json:"version"`
	Grants     []string `json:"grants"`
	Project    Project  `json:"project"`
}

// Project is the project the backend serves: one process per plugin and project.
type Project struct {
	ID string `json:"id"`
	// Root is the folder of the project on the pod; empty or unreachable over SSH (Local false):
	// a remote project is reached through the core (`fs.*`).
	Root  string `json:"root"`
	Local bool   `json:"local"`
}

// Granted says whether the user granted it (`files:read`, `net`, `core:fs.list`…).
func (i Info) Granted(grant string) bool {
	for _, g := range i.Grants {
		if g == grant {
			return true
		}
	}
	return false
}

// Method answers an `<id>.*` method of a view; its result is sent as JSON.
type Method func(ctx context.Context, params json.RawMessage) (any, error)

// Tool answers an assistant tool of the manifest (`contributes.toolSchemas`).
type Tool func(ctx context.Context, args json.RawMessage) (Result, error)

// Result is the answer of a tool: Content for the model, Summary a short line for the user,
// Error when the tool failed (the model reads Content as the reason).
type Result struct {
	Content string `json:"content"`
	Summary string `json:"summary,omitempty"`
	Error   bool   `json:"error,omitempty"`
}

// Plugin is what a backend serves.
type Plugin struct {
	// Init runs once at `initialize`, before any call; an error stops the backend.
	Init    func(ctx context.Context, info Info) error
	Methods map[string]Method
	Tools   map[string]Tool
	// Shutdown runs at `shutdown`, before the process exits.
	Shutdown func()
}

// Error is an error with a JSON-RPC code; any other error is sent with the code -32000.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// conn is one connection to the IDE.
type conn struct {
	p    Plugin
	wmu  sync.Mutex
	w    io.Writer
	next atomic.Int64
	mu   sync.Mutex
	wait map[string]chan message
	info Info
	// ready is closed once `initialize` is answered: calls wait for it, and get initErr when
	// Init failed.
	ready   chan struct{}
	initErr error
	once    sync.Once
	quit    chan struct{}
}

type ctxKey struct{}

func from(ctx context.Context) *conn {
	c, _ := ctx.Value(ctxKey{}).(*conn)
	return c
}

// InfoOf is the Info of the call's backend.
func InfoOf(ctx context.Context) Info {
	if c := from(ctx); c != nil {
		return c.info
	}
	return Info{}
}

// LangOf is the language of the window or conversation of a call or tool (`fr`, `es`…), empty
// for English.
func LangOf(ctx context.Context) string {
	l, _ := ctx.Value(langKey{}).(string)
	return l
}

type langKey struct{}

// Serve speaks to the IDE over stdin and stdout until `shutdown` or the end of stdin, then
// exits the process.
func Serve(p Plugin) {
	err := ServeIO(context.Background(), p, os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

// ServeIO is Serve over any streams; it returns at `shutdown` or the end of r.
func ServeIO(ctx context.Context, p Plugin, r io.Reader, w io.Writer) error {
	c := &conn{p: p, w: w, wait: map[string]chan message{}, ready: make(chan struct{}), quit: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.WithValue(ctx, ctxKey{}, c))
	defer cancel()
	lines := make(chan []byte)
	errc := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64<<10), MaxMessage)
		for sc.Scan() {
			lines <- append([]byte(nil), sc.Bytes()...)
		}
		errc <- sc.Err()
	}()
	for {
		select {
		case <-c.quit:
			return nil
		case err := <-errc:
			return err
		case line := <-lines:
			var m message
			if json.Unmarshal(line, &m) != nil {
				continue
			}
			if m.Method == "" {
				c.answered(m)
				continue
			}
			if m.Method == "initialize" {
				// Apart, since Init may call the core, whose answer comes through this loop.
				go c.initialize(ctx, m)
				continue
			}
			go c.handle(ctx, m)
		}
	}
}

func (c *conn) send(m message) error {
	m.JSONRPC = "2.0"
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if len(data) > MaxMessage {
		return fmt.Errorf("a message to the IDE is larger than %d MB", MaxMessage>>20)
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_, err = c.w.Write(append(data, '\n'))
	return err
}

func (c *conn) reply(id json.RawMessage, result any, err error) {
	if err == nil {
		var data []byte
		if data, err = json.Marshal(result); err == nil {
			_ = c.send(message{ID: id, Result: data})
			return
		}
	}
	var e *Error
	if !errors.As(err, &e) {
		e = &Error{Code: -32000, Message: err.Error()}
	}
	if err := c.send(message{ID: id, Error: e}); err != nil {
		// Too large, most likely: the error at least goes through.
		_ = c.send(message{ID: id, Error: &Error{Code: -32000, Message: err.Error()}})
	}
}

func (c *conn) initialize(ctx context.Context, m message) {
	var info Info
	err := json.Unmarshal(m.Params, &info)
	if err != nil {
		err = &Error{Code: -32602, Message: "invalid initialize: " + err.Error()}
	} else {
		c.info = info
		if c.p.Init != nil {
			err = c.p.Init(ctx, info)
		}
	}
	c.once.Do(func() {
		c.initErr = err
		close(c.ready)
	})
	c.reply(m.ID, map[string]any{"apiVersion": APIVersion}, err)
}

func (c *conn) handle(ctx context.Context, m message) {
	switch m.Method {
	case "shutdown":
		if c.p.Shutdown != nil {
			c.p.Shutdown()
		}
		c.reply(m.ID, map[string]any{}, nil)
		close(c.quit)
		return
	case "call", "tool":
	default:
		if m.ID != nil {
			c.reply(m.ID, nil, &Error{Code: -32601, Message: "unknown method: " + m.Method})
		}
		return
	}
	select {
	case <-c.ready:
	case <-ctx.Done():
		return
	}
	if c.initErr != nil {
		c.reply(m.ID, nil, c.initErr)
		return
	}
	var p struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
		Name   string          `json:"name"`
		Args   json.RawMessage `json:"args"`
		Lang   string          `json:"lang"`
	}
	if err := json.Unmarshal(m.Params, &p); err != nil {
		c.reply(m.ID, nil, &Error{Code: -32602, Message: "invalid " + m.Method + ": " + err.Error()})
		return
	}
	ctx = context.WithValue(ctx, langKey{}, p.Lang)
	if m.Method == "call" {
		f := c.p.Methods[p.Method]
		if f == nil {
			c.reply(m.ID, nil, &Error{Code: -32601, Message: "unknown method: " + p.Method})
			return
		}
		res, err := f(ctx, p.Params)
		c.reply(m.ID, res, err)
		return
	}
	f := c.p.Tools[p.Name]
	if f == nil {
		c.reply(m.ID, Result{Content: "unknown tool: " + p.Name, Error: true}, nil)
		return
	}
	res, err := f(ctx, p.Args)
	if err != nil {
		res = Result{Content: err.Error(), Summary: "failed", Error: true}
	}
	c.reply(m.ID, res, nil)
}

func (c *conn) answered(m message) {
	c.mu.Lock()
	ch := c.wait[string(m.ID)]
	delete(c.wait, string(m.ID))
	c.mu.Unlock()
	if ch != nil {
		ch <- m
	}
}

// Core calls a method of the core of the IDE in the project (`fs.list`, `fs.read`…), which the
// IDE refuses without the grant `core:<method>`; its result is decoded into out (nil: dropped).
func Core(ctx context.Context, method string, params, out any) error {
	c := from(ctx)
	if c == nil {
		return errors.New("backend.Core outside a call of Serve")
	}
	raw, err := json.Marshal(map[string]any{"method": method, "params": params})
	if err != nil {
		return err
	}
	id, _ := json.Marshal("core-" + strconv.FormatInt(c.next.Add(1), 10))
	ch := make(chan message, 1)
	c.mu.Lock()
	c.wait[string(id)] = ch
	c.mu.Unlock()
	forget := func() {
		c.mu.Lock()
		delete(c.wait, string(id))
		c.mu.Unlock()
	}
	if err := c.send(message{ID: id, Method: "core", Params: raw}); err != nil {
		forget()
		return err
	}
	select {
	case m := <-ch:
		if m.Error != nil {
			return m.Error
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(m.Result, out)
	case <-ctx.Done():
		forget()
		return ctx.Err()
	}
}

// Emit sends an event of the plugin (`<id>.something`) to the windows of the project, which
// its views hear with `ide.on`.
func Emit(ctx context.Context, event string, data any) error {
	c := from(ctx)
	if c == nil {
		return errors.New("backend.Emit outside a call of Serve")
	}
	raw, err := json.Marshal(map[string]any{"event": event, "data": data})
	if err != nil {
		return err
	}
	return c.send(message{Method: "emit", Params: raw})
}
