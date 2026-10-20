package main

import (
	"math/rand"
	"strings"
)

// Bot steers a computer player (or the autopilot). It looks again each time
// its player reaches a new cell.
type Bot struct {
	lastX, lastY int
	started      bool
}

var directionStep = map[PlayerDirection][2]int{
	PlayerUp:    {0, -1},
	PlayerLeft:  {-1, 0},
	PlayerDown:  {0, 1},
	PlayerRight: {1, 0},
}

var oppositeDirection = map[PlayerDirection]PlayerDirection{
	PlayerUp:    PlayerDown,
	PlayerLeft:  PlayerRight,
	PlayerDown:  PlayerUp,
	PlayerRight: PlayerLeft,
}

func (g *Game) botCount() int {
	n := 0
	for s := range g.sessions {
		if s.c == nil {
			n++
		}
	}
	return n
}

func colorHeldByBot(g *Game, name string) bool {
	for s := range g.sessions {
		if s.c == nil && strings.ToLower(playerColorNames[s.Player.Color]) == name {
			return true
		}
	}
	return false
}

// dropBot takes one computer player out, the one in the named colour if
// there is one, to make room for a person.
func (g *Game) dropBot(name string) {
	var drop *Session
	for s := range g.sessions {
		if s.c != nil {
			continue
		}
		if drop == nil || strings.ToLower(playerColorNames[s.Player.Color]) == name {
			drop = s
		}
	}
	if drop != nil {
		delete(g.sessions, drop)
	}
}

// balanceBots keeps up to want computer players in a game that has someone
// in it, as many as the free colours allow, and none in an empty one.
func (g *Game) balanceBots(want int) {
	if g.humanCount() == 0 {
		want = 0
	}
	for g.botCount() > want {
		g.dropBot("")
	}
	for g.botCount() < want {
		colors := g.availableColors()
		if len(colors) == 0 {
			return
		}
		s := NewSession(Conn{}, g.WorldWidth(), g.WorldHeight(),
			colors[rand.Intn(len(colors))])
		s.bot = &Bot{}
		g.sessions[s] = struct{}{}
	}
}

// occupied marks every cell a trail or a head is in
func (g *Game) occupied() [][]bool {
	w, h := g.WorldWidth(), g.WorldHeight()
	grid := make([][]bool, w)
	for x := range grid {
		grid[x] = make([]bool, h)
	}
	mark := func(x, y int) {
		if x >= 0 && x < w && y >= 0 && y < h {
			grid[x][y] = true
		}
	}
	for s := range g.sessions {
		p := s.Player
		mark(p.Pos.RoundX(), p.Pos.RoundY())
		for _, seg := range p.Trail {
			mark(seg.Pos.RoundX(), seg.Pos.RoundY())
		}
	}
	return grid
}

func (g *Game) steerBots() {
	var grid [][]bool
	for s := range g.sessions {
		if s.bot == nil {
			continue
		}
		p := s.Player
		x, y := p.Pos.RoundX(), p.Pos.RoundY()
		if s.bot.started && x == s.bot.lastX && y == s.bot.lastY {
			continue
		}
		s.bot.started = true
		s.bot.lastX, s.bot.lastY = x, y
		if grid == nil {
			grid = g.occupied()
		}
		s.bot.steer(p, grid)
	}
}

func (b *Bot) steer(p *Player, grid [][]bool) {
	w, h := len(grid), len(grid[0])
	free := func(x, y int) bool {
		return x >= 0 && x < w && y >= 0 && y < h && !grid[x][y]
	}
	x, y := p.Pos.RoundX(), p.Pos.RoundY()

	// how far each way is clear, and how much room lies beyond its first cell
	type option struct {
		dir  PlayerDirection
		run  int
		room int
	}
	var options []option
	for _, dir := range []PlayerDirection{PlayerUp, PlayerLeft, PlayerDown, PlayerRight} {
		if dir == oppositeDirection[p.InitDir] {
			continue
		}
		step := directionStep[dir]
		run := 0
		for cx, cy := x+step[0], y+step[1]; free(cx, cy) && run < 40; cx, cy = cx+step[0], cy+step[1] {
			run++
		}
		room := 0
		if run > 0 {
			room = roomFrom(grid, x+step[0], y+step[1], 400)
		}
		options = append(options, option{dir, run, room})
	}

	var current *option
	best := options[0]
	for i, o := range options {
		if o.dir == p.Direction {
			current = &options[i]
		}
		if o.room > best.room || (o.room == best.room && o.run > best.run) {
			best = o
		}
	}

	// A cell is about 10 ms wide and 14 ms tall at their speeds, so look
	// further ahead going up and down. Now and then a bot does not look at
	// all, which is how they crash.
	ahead := 3
	if p.Direction == PlayerUp || p.Direction == PlayerDown {
		ahead = 2
	}
	choice := p.Direction
	switch {
	case rand.Float64() < 0.03:
	case current == nil || current.run < ahead || current.room < best.room/2:
		choice = best.dir
	case rand.Float64() < 0.04:
		// a turn for its own sake, when there is room to make it
		turn := options[rand.Intn(len(options))]
		if turn.run >= ahead+2 && turn.room >= best.room/2 {
			choice = turn.dir
		}
	}

	switch choice {
	case PlayerUp:
		p.HandleUp()
	case PlayerLeft:
		p.HandleLeft()
	case PlayerDown:
		p.HandleDown()
	case PlayerRight:
		p.HandleRight()
	}
}

// roomFrom counts the free cells reachable from x, y, up to limit
func roomFrom(grid [][]bool, x, y, limit int) int {
	w, h := len(grid), len(grid[0])
	seen := map[[2]int]bool{{x, y}: true}
	queue := [][2]int{{x, y}}
	for len(queue) > 0 && len(seen) < limit {
		c := queue[0]
		queue = queue[1:]
		for _, d := range directionStep {
			n := [2]int{c[0] + d[0], c[1] + d[1]}
			if n[0] < 0 || n[0] >= w || n[1] < 0 || n[1] >= h || grid[n[0]][n[1]] || seen[n] {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	return len(seen)
}
