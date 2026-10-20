package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// serverConfig lets everyone in. Its host key is kept in the data folder,
// made the first time; a dedicated server still uses an id_rsa in the
// working directory if there is one, as upstream did.
func serverConfig(dedicated bool) (*ssh.ServerConfig, error) {
	// Everyone can login!
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	var private ssh.Signer
	if dedicated {
		if privateBytes, err := os.ReadFile("id_rsa"); err == nil {
			if private, err = ssh.ParsePrivateKey(privateBytes); err != nil {
				return nil, fmt.Errorf("id_rsa: %w", err)
			}
		}
	}
	if private == nil {
		var err error
		if private, err = hostKey(); err != nil {
			return nil, err
		}
	}

	config.AddHostKey(private)
	return config, nil
}

func hostKey() (ssh.Signer, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "host_key")
	if b, err := os.ReadFile(path); err == nil {
		return ssh.ParsePrivateKey(b)
	}

	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	block, err := ssh.MarshalPrivateKey(key, "sshtron host key")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func handler(conn net.Conn, gm *GameManager, config *ssh.ServerConfig,
	local, autopilot bool) {

	// Before use, a handshake must be performed on the incoming
	// net.Conn.
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake with new client: %v", err)
		return
	}
	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple
		// terminal interface.
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("could not accept channel: %v", err)
			return
		}

		log.Printf("Player joined from %s. Current stats: %d users, %d games",
			sshConn.RemoteAddr(), gm.SessionCount(), gm.GameCount())

		// Reject all out of band requests accept for the unix defaults, pty-req,
		// shell and window-change, which also tell the size of the terminal.
		size := newTermSize()
		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "pty-req":
					// string TERM, then uint32 columns and rows
					if len(req.Payload) >= 4 {
						n := binary.BigEndian.Uint32(req.Payload)
						if rest := req.Payload[4:]; uint32(len(rest)) >= n+8 {
							rest = rest[n:]
							size.set(int(binary.BigEndian.Uint32(rest)),
								int(binary.BigEndian.Uint32(rest[4:])))
						}
					}
					req.Reply(true, nil)
					continue
				case "window-change":
					if len(req.Payload) >= 8 {
						size.set(int(binary.BigEndian.Uint32(req.Payload)),
							int(binary.BigEndian.Uint32(req.Payload[4:])))
					}
					continue
				case "shell":
					req.Reply(true, nil)
					continue
				}
				req.Reply(false, nil)
			}
		}(requests)

		gm.HandleNewChannel(Conn{
			c:         channel,
			size:      size,
			local:     local,
			autopilot: autopilot,
		}, sshConn.User())
	}
}
