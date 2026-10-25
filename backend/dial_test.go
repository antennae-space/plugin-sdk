package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The IDE answers dial with an id, then sends the socket on fd 3 (here a socketpair standing
// for it); vault calls go through core.
func TestDialAndSecrets(t *testing.T) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		t.Fatal(err)
	}
	fd3 = os.NewFile(uintptr(pair[1]), "fd3")
	sockets = &fdInbox{waiting: map[int]chan *os.File{}, arrived: map[int]*os.File{}}
	podEnd, _ := net.FileConn(os.NewFile(uintptr(pair[0]), "pod"))
	defer podEnd.Close()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		line, _ := bufio.NewReader(c).ReadString('\n')
		fmt.Fprint(c, "echo "+line)
	}()

	vault := map[string]string{}
	var dialed map[string]any
	f := start(t, Plugin{
		Methods: map[string]Method{
			"db.ping": func(ctx context.Context, _ json.RawMessage) (any, error) {
				conn, err := Dial(ctx, "127.0.0.1", l.Addr().(*net.TCPAddr).Port, &SSH{Host: "bastion", User: "me", Auth: "agent"})
				if err != nil {
					return nil, err
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				fmt.Fprint(conn, "ping\n")
				return bufio.NewReader(conn).ReadString('\n')
			},
			"db.secret": func(ctx context.Context, _ json.RawMessage) (any, error) {
				if err := SetSecret(ctx, "pw", "s3cret"); err != nil {
					return nil, err
				}
				v, found, err := Secret(ctx, "pw")
				if err != nil || !found {
					return nil, fmt.Errorf("get: %v %v", found, err)
				}
				if err := DeleteSecret(ctx, "pw"); err != nil {
					return nil, err
				}
				_, found, _ = Secret(ctx, "pw")
				return fmt.Sprintf("%s %v", v, found), nil
			},
		},
	}, func(method string, params json.RawMessage) (any, *Error) {
		switch method {
		case "dial":
			_ = json.Unmarshal(params, &dialed)
			// The socket is sent after the answer, as the IDE may.
			go func() {
				time.Sleep(20 * time.Millisecond)
				c, _ := net.Dial("tcp", l.Addr().String())
				file, _ := c.(*net.TCPConn).File()
				c.Close()
				_, _, _ = podEnd.(*net.UnixConn).WriteMsgUnix([]byte("7"), syscall.UnixRights(int(file.Fd())), nil)
				file.Close()
			}()
			return map[string]any{"fd": 7}, nil
		case "vault.get":
			var p struct{ Key string }
			_ = json.Unmarshal(params, &p)
			v, ok := vault[p.Key]
			return map[string]any{"value": v, "found": ok}, nil
		case "vault.set":
			var p struct{ Key, Value string }
			_ = json.Unmarshal(params, &p)
			vault[p.Key] = p.Value
			return map[string]any{}, nil
		case "vault.delete":
			var p struct{ Key string }
			_ = json.Unmarshal(params, &p)
			delete(vault, p.Key)
			return map[string]any{}, nil
		}
		return nil, &Error{Code: -32000, Message: method + " is not granted"}
	})
	f.ask("initialize", initParams)

	m := f.ask("call", map[string]any{"method": "db.ping"})
	if m.Error != nil || !strings.Contains(string(m.Result), "echo ping") {
		t.Fatalf("dial: %s %v", m.Result, m.Error)
	}
	if ssh, _ := dialed["ssh"].(map[string]any); dialed["host"] != "127.0.0.1" || ssh["host"] != "bastion" || ssh["auth"] != "agent" {
		t.Fatalf("dial params: %v", dialed)
	}
	m = f.ask("call", map[string]any{"method": "db.secret"})
	if m.Error != nil || string(m.Result) != `"s3cret false"` {
		t.Fatalf("secrets: %s %v", m.Result, m.Error)
	}
}

func TestSettingsAndTargets(t *testing.T) {
	var declared json.RawMessage
	f := start(t, Plugin{Methods: map[string]Method{
		"db.init": func(ctx context.Context, _ json.RawMessage) (any, error) {
			var s struct{ Rows int }
			if err := Settings(ctx, &s); err != nil {
				return nil, err
			}
			return s.Rows, DeclareTargets(ctx, []Target{{ID: "main", Label: "Main DB"}})
		},
	}}, func(method string, params json.RawMessage) (any, *Error) {
		switch method {
		case "settings.get":
			return map[string]any{"rows": 42}, nil
		case "ai.targets":
			declared = params
			return map[string]any{}, nil
		}
		return nil, &Error{Code: -32000, Message: "no " + method}
	})
	f.ask("initialize", initParams)
	if m := f.ask("call", map[string]any{"method": "db.init"}); m.Error != nil || string(m.Result) != "42" {
		t.Fatalf("%s %v", m.Result, m.Error)
	}
	if string(declared) != `{"targets":[{"id":"main","label":"Main DB"}]}` {
		t.Fatalf("declared %s", declared)
	}
}

// A console the plugin feeds: its socket comes on fd 3, its resizes as notifications.
func TestTerminal(t *testing.T) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		t.Fatal(err)
	}
	fd3 = os.NewFile(uintptr(pair[1]), "fd3")
	sockets = &fdInbox{waiting: map[int]chan *os.File{}, arrived: map[int]*os.File{}}
	podEnd, _ := net.FileConn(os.NewFile(uintptr(pair[0]), "pod"))
	defer podEnd.Close()
	sizes := make(chan string, 1)
	var console net.Conn
	f := start(t, Plugin{
		Methods: map[string]Method{
			"x.shell": func(ctx context.Context, _ json.RawMessage) (any, error) {
				term, err := OpenTerminal(ctx, "Shell")
				if err != nil {
					return nil, err
				}
				term.OnResize(func(cols, rows int) { sizes <- fmt.Sprintf("%dx%d", cols, rows) })
				go func() { _, _ = term.Write([]byte("$ ")) }()
				return term.ID, nil
			},
		},
	}, func(method string, params json.RawMessage) (any, *Error) {
		if method != "terminal.open" {
			return nil, &Error{Code: -32000, Message: "no " + method}
		}
		fds, _ := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		console, _ = net.FileConn(os.NewFile(uintptr(fds[0]), "console"))
		theirs := os.NewFile(uintptr(fds[1]), "theirs")
		_, _, _ = podEnd.(*net.UnixConn).WriteMsgUnix([]byte("4"), syscall.UnixRights(int(theirs.Fd())), nil)
		theirs.Close()
		return map[string]any{"console": "c1", "fd": 4}, nil
	})
	f.ask("initialize", initParams)
	if m := f.ask("call", map[string]any{"method": "x.shell"}); string(m.Result) != `"c1"` {
		t.Fatalf("opened: %s %v", m.Result, m.Error)
	}
	b := make([]byte, 2)
	_ = console.SetReadDeadline(time.Now().Add(5 * time.Second))
	if n, _ := console.Read(b); string(b[:n]) != "$ " {
		t.Fatalf("its output: %q", b[:n])
	}
	f.write(message{JSONRPC: "2.0", Method: "terminal.resize", Params: mustJSON(map[string]any{"console": "c1", "cols": 90, "rows": 20})})
	select {
	case got := <-sizes:
		if got != "90x20" {
			t.Fatalf("size %s", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no resize")
	}
}
