package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// app is the player's side: the terminal, the menu, a game hosted here and
// the SSH client that joins games, here or elsewhere.
type app struct {
	state  *term.State
	keys   chan []byte
	colour string

	// the game hosted here, started the first time it is needed
	gm       *GameManager
	config   *ssh.ServerConfig
	listener net.Listener
}

func newApp(colour string) (*app, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, errors.New("needs a terminal (for a server, use -server)")
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	enableVT()
	a := &app{state: state, keys: make(chan []byte, 16), colour: colour}
	// the alternate screen, without a cursor
	os.Stdout.WriteString("\033[?1049h\033[?25l")

	// one reader for the whole run, so the menu and the games take turns
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				a.keys <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				close(a.keys)
				return
			}
		}
	}()
	return a, nil
}

func (a *app) close() {
	if a.state == nil {
		return
	}
	os.Stdout.WriteString("\033[0m\033[?25h\033[?1049l")
	term.Restore(int(os.Stdin.Fd()), a.state)
	a.state = nil
	if a.listener != nil {
		a.listener.Close()
	}
}

// startHost starts the game hosted here, open to the network on the first
// free port from port on; if none is free, it is only for this player.
func (a *app) startHost(port, bots int) error {
	if a.gm != nil {
		return nil
	}
	config, err := serverConfig(false)
	if err != nil {
		return err
	}
	a.config = config
	a.gm = NewGameManager(bots)

	for p := port; p < port+10; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", p))
		if err != nil {
			continue
		}
		a.listener = l
		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				go handler(conn, a.gm, a.config, false, false)
			}
		}()
		break
	}
	return nil
}

// host plays in the game hosted here, over a connection of its own on the
// loopback interface (both ends of an SSH handshake write at once, which a
// net.Pipe cannot take)
func (a *app) host(port, bots int, autopilot bool) error {
	if err := a.startHost(port, bots); err != nil {
		return err
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	go func() {
		defer l.Close()
		if conn, err := l.Accept(); err == nil {
			handler(conn, a.gm, a.config, true, autopilot)
		}
	}()
	client, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		l.Close()
		return err
	}
	return a.play(client, "here", ssh.InsecureIgnoreHostKey())
}

// join plays on a server elsewhere. Its host key is remembered the first
// time and checked after that, as ssh does.
func (a *app) join(addr string) error {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		port := defaultSshPort
		if addr == publicServer {
			port = 22
		}
		addr = net.JoinHostPort(addr, fmt.Sprint(port))
	}

	callback, err := a.knownHosts()
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	return a.play(conn, addr, callback)
}

func (a *app) knownHosts() (ssh.HostKeyCallback, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "known_hosts")
	if f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND, 0o600); err == nil {
		f.Close()
	}
	check, err := knownhosts.New(path)
	if err != nil {
		return nil, err
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := check(hostname, remote, key)
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			f, ferr := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
			if ferr != nil {
				return ferr
			}
			defer f.Close()
			_, ferr = fmt.Fprintln(f, knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key))
			return ferr
		}
		if errors.As(err, &keyErr) {
			return fmt.Errorf("%s has a different key from last time; "+
				"if that is expected, remove its line from %s", hostname, path)
		}
		return err
	}, nil
}

// play runs one game over conn until the server ends it
func (a *app) play(conn net.Conn, addr string, callback ssh.HostKeyCallback) error {
	user := strings.ToLower(a.colour)
	if user == "" {
		user = "player"
	}
	cc, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: callback,
		Timeout:         10 * time.Second,
	})
	if err != nil {
		conn.Close()
		return err
	}
	client := ssh.NewClient(cc, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cols, rows := windowSize()
	if err := session.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{}); err != nil {
		return err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	session.Stdout = os.Stdout
	session.Stderr = io.Discard
	os.Stdout.WriteString("\033[0m\033[2J\033[H")
	if err := session.Shell(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	resize := time.NewTicker(250 * time.Millisecond)
	defer resize.Stop()

	for {
		select {
		case <-done:
			os.Stdout.WriteString("\033[0m\033[?25l")
			return nil
		case <-resize.C:
			if c, r := windowSize(); c != cols || r != rows {
				cols, rows = c, r
				session.WindowChange(rows, cols)
			}
		case b, ok := <-a.keys:
			if !ok {
				return nil
			}
			stdin.Write(steeringKeys(b))
		}
	}
}

// steeringKeys turns arrow keys into WASD, which every SSHTron server
// knows; upstream's takes the Esc an arrow key starts with for quitting.
func steeringKeys(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == 27 && i+2 < len(b) && (b[i+1] == '[' || b[i+1] == 'O') {
			if k, ok := map[byte]byte{'A': 'w', 'B': 's', 'C': 'd', 'D': 'a'}[b[i+2]]; ok {
				out = append(out, k)
				i += 2
				continue
			}
		}
		out = append(out, b[i])
	}
	return out
}

func windowSize() (int, int) {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return cols, rows
}
