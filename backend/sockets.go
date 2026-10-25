package backend

import (
	"context"
	"encoding/json"
	"net"
	"sync"
)

// Socket asks the IDE to open a named socket for the plugin and returns the connection: "docker"
// (the grant socket:docker, as much as root on the host: the IDE says so when it asks). On an SSH
// project it is the socket of the project's host.
func Socket(ctx context.Context, name string) (net.Conn, error) {
	var r struct {
		FD int `json:"fd"`
	}
	if err := Core(ctx, "socket", map[string]string{"name": name}, &r); err != nil {
		return nil, err
	}
	return sockets.take(r.FD)
}

// Terminal is a console of the IDE the plugin feeds (the grant terminal): what it writes shows
// there, what the user types there is read from it; Close ends it.
type Terminal struct {
	net.Conn
	// ID is the console's: the views show it with ide.showConsole(id).
	ID string
}

// OnResize is told each new size of the console (its columns and rows).
func (t *Terminal) OnResize(f func(cols, rows int)) {
	resizeMu.Lock()
	resizers[t.ID] = f
	resizeMu.Unlock()
}

var (
	resizeMu sync.Mutex
	resizers = map[string]func(cols, rows int){}
)

func resized(params json.RawMessage) {
	var r struct {
		Console    string
		Cols, Rows int
	}
	if json.Unmarshal(params, &r) != nil {
		return
	}
	resizeMu.Lock()
	f := resizers[r.Console]
	resizeMu.Unlock()
	if f != nil {
		f(r.Cols, r.Rows)
	}
}

// OpenTerminal opens a console of the IDE titled title, fed by the plugin (docker exec, a REPL):
// the IDE draws it, the plugin gives the bytes.
func OpenTerminal(ctx context.Context, title string) (*Terminal, error) {
	var r struct {
		Console string `json:"console"`
		FD      int    `json:"fd"`
	}
	if err := Core(ctx, "terminal.open", map[string]string{"title": title}, &r); err != nil {
		return nil, err
	}
	conn, err := sockets.take(r.FD)
	if err != nil {
		return nil, err
	}
	return &Terminal{Conn: conn, ID: r.Console}, nil
}
