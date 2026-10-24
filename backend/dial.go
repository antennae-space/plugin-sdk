package backend

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
)

// SSH is the tunnel a connection goes through, opened by the IDE's SSH pool with the user's
// agent and keys (the grant ssh). Auth: agent, key (KeyPath in ~/.ssh) or password.
type SSH struct {
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"`
	User       string `json:"user"`
	Auth       string `json:"auth"`
	KeyPath    string `json:"keyPath,omitempty"`
	Password   string `json:"password,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// Dial asks the IDE to connect to host:port for the plugin (the grant dial; through ssh when
// given) and returns the connection. The user allows each new destination once: the call waits
// for their answer, and fails when they deny it. A confined backend cannot open the network
// itself without the grant net; this is the way to reach what the user chooses.
func Dial(ctx context.Context, host string, port int, ssh *SSH) (net.Conn, error) {
	params := map[string]any{"host": host, "port": port}
	if ssh != nil {
		params["ssh"] = ssh
	}
	var r struct {
		FD int `json:"fd"`
	}
	if err := Core(ctx, "dial", params, &r); err != nil {
		return nil, err
	}
	return sockets.take(r.FD)
}

// sockets are those the IDE sends on fd 3, by the id its dial answers; one may come before the
// answer that names it, or after another one.
var sockets = &fdInbox{waiting: map[int]chan *os.File{}, arrived: map[int]*os.File{}}

type fdInbox struct {
	once    sync.Once
	err     error
	mu      sync.Mutex
	waiting map[int]chan *os.File
	arrived map[int]*os.File
}

// fd3 is kept for the life of the process: an *os.File closes its fd when collected.
var fd3 = os.NewFile(3, "ide-sockets")

func (in *fdInbox) take(id int) (net.Conn, error) {
	in.once.Do(func() { in.err = in.listen() })
	if in.err != nil {
		return nil, in.err
	}
	in.mu.Lock()
	f, ok := in.arrived[id]
	var ch chan *os.File
	if ok {
		delete(in.arrived, id)
	} else {
		ch = make(chan *os.File, 1)
		in.waiting[id] = ch
	}
	in.mu.Unlock()
	if !ok {
		f = <-ch
	}
	if f == nil {
		return nil, errors.New("the channel of the IDE's sockets closed")
	}
	defer f.Close()
	return net.FileConn(f)
}

// listen reads fd 3 for the life of the process.
func (in *fdInbox) listen() error {
	if fd3 == nil {
		return errors.New("no fd 3: not started by the IDE")
	}
	c, err := net.FileConn(fd3)
	if err != nil {
		return fmt.Errorf("fd 3 is not the IDE's socket channel: %v", err)
	}
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return errors.New("fd 3 is not the IDE's socket channel")
	}
	go func() {
		buf, oob := make([]byte, 32), make([]byte, syscall.CmsgSpace(4))
		for {
			n, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
			if err != nil || n == 0 {
				in.closeAll()
				return
			}
			id, _ := strconv.Atoi(string(buf[:n]))
			msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
			if err != nil || len(msgs) != 1 {
				continue
			}
			fds, err := syscall.ParseUnixRights(&msgs[0])
			if err != nil || len(fds) != 1 {
				continue
			}
			f := os.NewFile(uintptr(fds[0]), "ide-socket")
			in.mu.Lock()
			if ch, ok := in.waiting[id]; ok {
				delete(in.waiting, id)
				ch <- f
			} else {
				in.arrived[id] = f
			}
			in.mu.Unlock()
		}
	}()
	return nil
}

func (in *fdInbox) closeAll() {
	in.mu.Lock()
	defer in.mu.Unlock()
	for id, ch := range in.waiting {
		close(ch)
		delete(in.waiting, id)
	}
}
