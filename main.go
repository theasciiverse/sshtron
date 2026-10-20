package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/fatih/color"
)

const (
	sshPortEnv     = "SSH_PORT"
	defaultSshPort = 2022

	publicServer = "sshtron.zachlatta.com"
)

var version = "dev"

func main() {
	var (
		server  = flag.Bool("server", false, "run only the server, for others to join with ssh")
		play    = flag.Bool("play", false, "skip the menu: host a game here and play in it")
		join    = flag.String("join", "", "skip the menu: join the game at `host[:port]`")
		online  = flag.Bool("online", false, "skip the menu: join the public server, "+publicServer)
		demo    = flag.Bool("demo", false, "host a game and watch the computer play; any key ends it")
		port    = flag.Int("port", 0, "SSH port to host on (default 2022, or $SSH_PORT)")
		bots    = flag.Int("bots", -1, "computer players in each game (default 3, none with -server)")
		colour  = flag.String("color", "", "colour to ask for: red, green, yellow, blue, magenta, cyan or white")
		showVer = flag.Bool("version", false, "print the version")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: sshtron [options]\n\n"+
			"With no options, a menu: play here against the computer (friends on\n"+
			"your network can join), play on the public server, or join a friend.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVer {
		fmt.Println("sshtron", version)
		return
	}

	// the colours are sent to the players, whatever this process's own
	// output is
	color.NoColor = false

	sshPort := *port
	if sshPort == 0 {
		sshPort = defaultSshPort
		if p, err := strconv.Atoi(os.Getenv(sshPortEnv)); err == nil {
			sshPort = p
		}
	}

	if *server {
		if *bots < 0 {
			*bots = 0
		}
		runServer(sshPort, *bots)
		return
	}

	if *bots < 0 {
		*bots = 3
	}
	if *bots > len(playerColors)-1 {
		*bots = len(playerColors) - 1
	}

	// The server's log would land in the middle of the game, so it goes to
	// $SSHTRON_LOG or nowhere.
	log.SetOutput(io.Discard)
	if path := os.Getenv("SSHTRON_LOG"); path != "" {
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			log.SetOutput(f)
		}
	}

	app, err := newApp(*colour)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sshtron:", err)
		os.Exit(1)
	}
	defer app.close()

	switch {
	case *join != "":
		err = app.join(*join)
	case *online:
		err = app.join(publicServer)
	case *play, *demo:
		err = app.host(sshPort, *bots, *demo)
	default:
		app.menu(sshPort, *bots)
	}
	app.close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "sshtron:", err)
		os.Exit(1)
	}
}

// runServer is the upstream server: games for anyone who connects, until it
// is stopped.
func runServer(port, bots int) {
	config, err := serverConfig(true)
	if err != nil {
		log.Fatal(err)
	}

	gm := NewGameManager(bots)

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on port %d for SSH...", port)

	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handler(nConn, gm, config, false, false)
	}
}
