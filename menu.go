package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
)

const (
	itemPlay = iota
	itemOnline
	itemJoin
	itemColour
	itemQuit
)

var menuItems = []string{
	"Play",
	"Play online",
	"Join a friend",
	"Colour",
	"Quit",
}

var menuNotes = []string{
	"a game here against the computer",
	publicServer,
	"type the address of their game",
	"",
	"",
}

// menu is the start screen, and where a game returns to
func (a *app) menu(port, bots int) {
	// the hosted game opens at once, so friends can join it from the start
	status := ""
	if err := a.startHost(port, bots); err != nil {
		status = err.Error()
	}

	selected := itemPlay
	editing := false
	address := ""

	for {
		a.drawMenu(selected, editing, address, status)
		b, ok := <-a.keys
		if !ok {
			return
		}

		for _, key := range splitKeys(b) {
			if editing {
				switch key {
				case "enter":
					editing = false
					if strings.TrimSpace(address) == "" {
						continue
					}
					status = ""
					if err := a.join(strings.TrimSpace(address)); err != nil {
						status = "Could not join " + address + ": " + err.Error()
					}
				case "esc":
					editing = false
				case "backspace":
					if address != "" {
						_, n := utf8.DecodeLastRuneInString(address)
						address = address[:len(address)-n]
					}
				default:
					if utf8.RuneCountInString(key) == 1 && key[0] >= ' ' && len(address) < 60 {
						address += key
					}
				}
				continue
			}

			switch key {
			case "up", "w", "k":
				selected = (selected + len(menuItems) - 1) % len(menuItems)
			case "down", "s", "j", "tab":
				selected = (selected + 1) % len(menuItems)
			case "left", "a", "h":
				if selected == itemColour {
					a.colour = nextColour(a.colour, -1)
				}
			case "right", "d", "l":
				if selected == itemColour {
					a.colour = nextColour(a.colour, 1)
				}
			case "esc", "q", "ctrl-c":
				return
			case "enter", " ":
				status = ""
				switch selected {
				case itemPlay:
					if err := a.host(port, bots, false); err != nil {
						status = err.Error()
					}
				case itemOnline:
					if err := a.join(publicServer); err != nil {
						status = "Could not reach " + publicServer + ": " + err.Error()
					}
				case itemJoin:
					editing = true
				case itemColour:
					a.colour = nextColour(a.colour, 1)
				case itemQuit:
					return
				}
			}
		}
	}
}

func nextColour(current string, step int) string {
	names := []string{""}
	for _, c := range playerColors {
		names = append(names, strings.ToLower(playerColorNames[c]))
	}
	i := 0
	for j, n := range names {
		if n == strings.ToLower(current) {
			i = j
		}
	}
	return names[(i+step+len(names))%len(names)]
}

var titleArt = []string{
	"╭─╴╭─╴╷ ╷╶┬╴┌─╮╭─╮┌╮╷",
	"╰─╮╰─╮├─┤ │ ├┬╯│ ││╰┤",
	"╶─╯╶─╯╵ ╵ ╵ ╵╰╴╰─╯╵ ╵",
}

func (a *app) drawMenu(selected int, editing bool, address, status string) {
	cols, rows := windowSize()
	var lines []string
	add := func(s string) { lines = append(lines, s) }

	cyan := color.New(color.FgHiCyan).SprintFunc()
	dim := color.New(color.FgHiBlack).SprintFunc()
	bright := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	// the title lines are centred on their own
	centred := map[int]bool{}
	for _, l := range titleArt {
		centred[len(lines)] = true
		add(cyan(l))
	}
	add("")
	centred[len(lines)] = true
	add(dim("lightcycles over SSH"))
	add("")

	for i, item := range menuItems {
		label := item
		note := menuNotes[i]
		switch i {
		case itemColour:
			name := "any"
			c := color.New(color.FgHiWhite)
			for _, pc := range playerColors {
				if strings.ToLower(playerColorNames[pc]) == a.colour {
					name = playerColorNames[pc]
					c = color.New(pc, color.Bold)
				}
			}
			label = fmt.Sprintf("Colour  ◂ %s ▸", c.Sprint(name))
			note = ""
		case itemJoin:
			if editing {
				label = "Join  " + bright(address) + "█"
				note = "Enter joins, Esc goes back"
			}
		}
		if pad := 15 - visibleWidth(label); pad > 0 && note != "" {
			label += strings.Repeat(" ", pad)
		}
		if i == selected {
			add(bright("▸ ") + bright(label) + dim(note))
		} else {
			add("  " + label + dim(note))
		}
	}

	add("")
	if a.listener != nil {
		port := a.listener.Addr().(*net.TCPAddr).Port
		add(dim("Friends on your network join your game with"))
		for _, ip := range localAddresses() {
			add(dim(fmt.Sprintf("  sshtron -join %s:%d   or   ssh -p %d %s", ip, port, port, ip)))
		}
	}
	add("")
	add(dim("In a game: WASD, arrows or hjkl to steer, Esc to leave"))
	if status != "" {
		add("")
		add(color.New(color.FgHiRed).Sprint(status))
	}

	// centre the block on its widest line
	width := 0
	for _, l := range lines {
		if w := visibleWidth(l); w > width {
			width = w
		}
	}
	left := (cols - width) / 2
	if left < 0 {
		left = 0
	}
	top := (rows - len(lines)) / 2
	if top < 0 {
		top = 0
	}

	var b strings.Builder
	b.WriteString("\033[0m\033[?25l\033[2J")
	for i, l := range lines {
		if top+i >= rows {
			break
		}
		x := left
		if centred[i] {
			x = left + (width-visibleWidth(l))/2
		}
		fmt.Fprintf(&b, "\033[%d;%dH%s", top+i+1, x+1, l)
	}
	os.Stdout.WriteString(b.String())
}

// visibleWidth counts the runes outside escape codes
func visibleWidth(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc:
			if r >= '@' && r <= '~' && r != '[' {
				inEsc = false
			}
		case r == 27:
			inEsc = true
		default:
			n++
		}
	}
	return n
}

// splitKeys names the keys in one read from the terminal
func splitKeys(b []byte) []string {
	var keys []string
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c == 27 && i+2 < len(b) && (b[i+1] == '[' || b[i+1] == 'O'):
			name := map[byte]string{'A': "up", 'B': "down", 'C': "right", 'D': "left"}[b[i+2]]
			i += 2
			// skip the rest of a longer sequence
			for i < len(b) && (b[i] < '@' || b[i] > '~') {
				i++
			}
			if name != "" {
				keys = append(keys, name)
			}
		case c == 27:
			keys = append(keys, "esc")
		case c == '\r' || c == '\n':
			keys = append(keys, "enter")
		case c == 127 || c == 8:
			keys = append(keys, "backspace")
		case c == 3:
			keys = append(keys, "ctrl-c")
		case c == '\t':
			keys = append(keys, "tab")
		case c >= 0x80:
			_, n := utf8.DecodeRune(b[i:])
			keys = append(keys, string(b[i:i+n]))
			i += n - 1
		default:
			keys = append(keys, string(c))
		}
	}
	return keys
}

// localAddresses lists this machine's IPv4 addresses on the local network
func localAddresses() []string {
	var out []string
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// container and virtual machine bridges are not the local network
		if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "br-") ||
			strings.HasPrefix(iface.Name, "veth") || strings.HasPrefix(iface.Name, "virbr") {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok && ipnet.IP.To4() != nil && ipnet.IP.IsPrivate() {
				out = append(out, ipnet.IP.String())
			}
		}
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return out
}
