package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dustinkirkland/golang-petname"
	"github.com/fatih/color"
)

type Position struct {
	X float64
	Y float64
}

func PositionFromInt(x, y int) Position {
	return Position{float64(x), float64(y)}
}

func (p Position) RoundX() int {
	return int(p.X + 0.5)
}

func (p Position) RoundY() int {
	return int(p.Y + 0.5)
}

type PlayerDirection int

const (
	speedMultiplier            = 1
	verticalPlayerSpeed        = 0.007 * speedMultiplier
	horizontalPlayerSpeed      = 0.01 * speedMultiplier
	playerCountScoreMultiplier = 1.25
	playerTimeout              = 15 * time.Second

	//playerUpRune    = '⇡'
	//playerLeftRune  = '⇠'
	//playerDownRune  = '⇣'
	//playerRightRune = '⇢'
	playerUpRune    = '↑'
	playerLeftRune  = '←'
	playerDownRune  = '↓'
	playerRightRune = '→'

	//playerTrailHorizontal      = '╌'
	//playerTrailVertical        = '┆'
	playerTrailHorizontal = '─'
	playerTrailVertical   = '│'

	playerTrailLeftCornerUp    = '╭'
	playerTrailLeftCornerDown  = '╰'
	playerTrailRightCornerDown = '╯'
	playerTrailRightCornerUp   = '╮'

	playerRed     = color.FgRed
	playerGreen   = color.FgGreen
	playerYellow  = color.FgYellow
	playerBlue    = color.FgBlue
	playerMagenta = color.FgMagenta
	playerCyan    = color.FgCyan
	playerWhite   = color.FgWhite

	PlayerUp PlayerDirection = iota
	PlayerLeft
	PlayerDown
	PlayerRight
)

var playerColors = []color.Attribute{
	playerRed, playerGreen, playerYellow, playerBlue,
	playerMagenta, playerCyan, playerWhite,
}

var playerBorderColors = map[color.Attribute]color.Attribute{
	playerRed:     color.FgHiRed,
	playerGreen:   color.FgHiGreen,
	playerYellow:  color.FgHiYellow,
	playerBlue:    color.FgHiBlue,
	playerMagenta: color.FgHiMagenta,
	playerCyan:    color.FgHiCyan,
	playerWhite:   color.FgHiWhite,
}

var playerColorNames = map[color.Attribute]string{
	playerRed:     "Red",
	playerGreen:   "Green",
	playerYellow:  "Yellow",
	playerBlue:    "Blue",
	playerMagenta: "Magenta",
	playerCyan:    "Cyan",
	playerWhite:   "White",
}

type PlayerTrailSegment struct {
	Marker rune
	Pos    Position
	Color  color.Attribute
}

type Player struct {
	s *Session

	Name      string
	CreatedAt time.Time
	Direction PlayerDirection
	InitDir   PlayerDirection
	Marker    rune
	Color     color.Attribute
	Pos       *Position

	Trail []PlayerTrailSegment

	score float64
}

// NewPlayer creates a new player. If color is below 1, a random color is chosen
func NewPlayer(s *Session, worldWidth, worldHeight int,
	color color.Attribute) *Player {

	rand.Seed(time.Now().UnixNano())

	startX := rand.Float64() * float64(worldWidth)
	startY := rand.Float64() * float64(worldHeight)

	if color < 0 {
		color = playerColors[rand.Intn(len(playerColors))]
	}

	return &Player{
		s:         s,
		CreatedAt: time.Now(),
		Marker:    playerDownRune,
		Direction: PlayerDown,
		InitDir:   PlayerDown,
		Color:     color,
		Pos:       &Position{startX, startY},
	}
}

func (p *Player) addTrailSegment(pos Position, marker rune) {
	segment := PlayerTrailSegment{marker, pos, p.Color}
	p.Trail = append([]PlayerTrailSegment{segment}, p.Trail...)
}

func (p *Player) calculateScore(delta float64, playerCount int) float64 {
	rawIncrement := (delta * (float64(playerCount-1) * playerCountScoreMultiplier))

	// Convert millisecond increment to seconds
	actualIncrement := rawIncrement / 1000

	return p.score + actualIncrement
}

func (p *Player) HandleUp() {
	if p.InitDir == PlayerDown {
		return
	}
	p.Direction = PlayerUp
	p.s.didAction()
}

func (p *Player) HandleLeft() {
	if p.InitDir == PlayerRight {
		return
	}
	p.Direction = PlayerLeft
	p.s.didAction()
}

func (p *Player) HandleDown() {
	if p.InitDir == PlayerUp {
		return
	}
	p.Direction = PlayerDown
	p.s.didAction()
}

func (p *Player) HandleRight() {
	if p.InitDir == PlayerLeft {
		return
	}
	p.Direction = PlayerRight
	p.s.didAction()
}

func (p *Player) Score() int {
	return int(p.score)
}

func (p *Player) Update(g *Game, delta float64) {
	startX, startY := p.Pos.RoundX(), p.Pos.RoundY()

	switch p.Direction {
	case PlayerUp:
		p.Pos.Y -= verticalPlayerSpeed * delta
	case PlayerLeft:
		p.Pos.X -= horizontalPlayerSpeed * delta
	case PlayerDown:
		p.Pos.Y += verticalPlayerSpeed * delta
	case PlayerRight:
		p.Pos.X += horizontalPlayerSpeed * delta
	}

	endX, endY := p.Pos.RoundX(), p.Pos.RoundY()

	// If we moved, add a trail segment.
	if endX != startX || endY != startY {
		// Update initial direction
		// Needed to prevent players from doing a 180 in one frame
		p.InitDir = p.Direction
		// Update player direction only after a new frame
		switch p.InitDir {
		case PlayerUp:
			p.Marker = playerUpRune
		case PlayerLeft:
			p.Marker = playerLeftRune
		case PlayerDown:
			p.Marker = playerDownRune
		case PlayerRight:
			p.Marker = playerRightRune
		}

		var lastSeg *PlayerTrailSegment
		var lastSegX, lastSegY int
		if len(p.Trail) > 0 {
			lastSeg = &p.Trail[0]
			lastSegX = lastSeg.Pos.RoundX()
			lastSegY = lastSeg.Pos.RoundY()
		}

		pos := PositionFromInt(startX, startY)

		switch {
		// Handle corners. This took an ungodly amount of time to figure out. Highly
		// recommend you don't touch.
		case lastSeg != nil &&
			(p.Direction == PlayerRight && endX > lastSegX && endY < lastSegY) ||
			(p.Direction == PlayerDown && endX < lastSegX && endY > lastSegY):
			p.addTrailSegment(pos, playerTrailLeftCornerUp)
		case lastSeg != nil &&
			(p.Direction == PlayerUp && endX > lastSegX && endY < lastSegY) ||
			(p.Direction == PlayerLeft && endX < lastSegX && endY > lastSegY):
			p.addTrailSegment(pos, playerTrailRightCornerDown)
		case lastSeg != nil &&
			(p.Direction == PlayerDown && endX > lastSegX && endY > lastSegY) ||
			(p.Direction == PlayerLeft && endX < lastSegX && endY < lastSegY):
			p.addTrailSegment(pos, playerTrailRightCornerUp)
		case lastSeg != nil &&
			(p.Direction == PlayerRight && endX > lastSegX && endY > lastSegY) ||
			(p.Direction == PlayerUp && endX < lastSegX && endY < lastSegY):
			p.addTrailSegment(pos, playerTrailLeftCornerDown)

		// Vertical and horizontal trails
		case endX == startX && endY < startY:
			p.addTrailSegment(pos, playerTrailVertical)
		case endX < startX && endY == startY:
			p.addTrailSegment(pos, playerTrailHorizontal)
		case endX == startX && endY > startY:
			p.addTrailSegment(pos, playerTrailVertical)
		case endX > startX && endY == startY:
			p.addTrailSegment(pos, playerTrailHorizontal)
		}
	}

	p.score = p.calculateScore(delta, len(g.players()))
}

type ByColor []*Player

func (slice ByColor) Len() int {
	return len(slice)
}

func (slice ByColor) Less(i, j int) bool {
	return playerColorNames[slice[i].Color] < playerColorNames[slice[j].Color]
}

func (slice ByColor) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type TileType int

const (
	TileGrass TileType = iota
	TileBlocker
)

type Tile struct {
	Type TileType
}

const (
	gameWidth  = 78
	gameHeight = 22

	// frames sent to each player per second; the game itself updates at 60
	framesPerSecond = 20

	keyW = 'w'
	keyA = 'a'
	keyS = 's'
	keyD = 'd'

	keyZ = 'z'
	keyQ = 'q'
	// keyS and keyD are already defined

	keyH = 'h'
	keyJ = 'j'
	keyK = 'k'
	keyL = 'l'

	keyComma = ','
	keyO     = 'o'
	keyE     = 'e'

	keyCtrlC  = 3
	keyEscape = 27
)

// GameManager holds the running games. Its lock is always taken before a
// game's own lock.
type GameManager struct {
	mu    sync.Mutex
	Games map[string]*Game

	// computer players each game keeps (fewer when humans need the colours)
	Bots int
}

func NewGameManager(bots int) *GameManager {
	return &GameManager{
		Games: map[string]*Game{},
		Bots:  bots,
	}
}

// getGameWithAvailability returns a reference to a game with available spots for
// players. If one does not exist, nil is returned. A colour held by a computer
// player counts as available. Games whose last player has left are dropped.
func (gm *GameManager) getGameWithAvailability() *Game {
	var g *Game

	for name, game := range gm.Games {
		game.mu.Lock()
		closed := game.closed
		spots := len(game.availableColors()) + game.botCount()
		game.mu.Unlock()
		if closed {
			delete(gm.Games, name)
			continue
		}
		if spots > 0 && g == nil {
			g = game
		}
	}

	return g
}

func (gm *GameManager) SessionCount() int {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	sum := 0
	for _, game := range gm.Games {
		game.mu.Lock()
		sum += game.humanCount()
		game.mu.Unlock()
	}
	return sum
}

func (gm *GameManager) GameCount() int {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return len(gm.Games)
}

// Conn is one player's side of the game: an SSH channel and what its client
// said about the terminal.
type Conn struct {
	c    io.ReadWriteCloser
	size *termSize

	// local is the host's own player, who is never timed out
	local bool
	// autopilot steers the player like a computer player; any key ends it
	autopilot bool
}

func (gm *GameManager) HandleNewChannel(conn Conn, color string) {
	gm.mu.Lock()
	g := gm.getGameWithAvailability()
	if g == nil {
		g = NewGame(gameWidth, gameHeight)
		gm.Games[g.Name] = g

		go g.Run()
	}

	g.mu.Lock()
	color = strings.ToLower(color)
	colorOptions := g.availableColors()
	if len(colorOptions) == 0 || colorHeldByBot(g, color) {
		g.dropBot(color)
		colorOptions = g.availableColors()
	}
	finalColor := colorOptions[0]

	// choose the requested color if available
	for _, clr := range colorOptions {
		if strings.ToLower(playerColorNames[clr]) == color {
			finalColor = clr
			break
		}
	}

	session := NewSession(conn, g.WorldWidth(), g.WorldHeight(), finalColor)
	if conn.autopilot {
		session.bot = &Bot{}
	}
	g.addSession(session)
	g.balanceBots(gm.Bots)
	g.mu.Unlock()
	gm.mu.Unlock()

	go func() {
		reader := bufio.NewReader(conn.c)
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				log.Printf("%s: %v", playerColorNames[finalColor], err)
				break
			}
			if r == keyEscape {
				// an arrow key is Esc [ or Esc O and a letter, sent together
				if reader.Buffered() > 1 {
					r1, _, _ := reader.ReadRune()
					r2, _, _ := reader.ReadRune()
					if r1 == '[' || r1 == 'O' {
						r = map[rune]rune{'A': keyW, 'B': keyS, 'C': keyD, 'D': keyA}[r2]
					} else {
						r = 0
					}
				}
			}
			log.Printf("%s: key %q", playerColorNames[finalColor], r)

			if session.autopilot && r != 0 {
				r = keyEscape
			}

			g.mu.Lock()
			switch r {
			case keyW, keyZ, keyK, keyComma, 'W', 'Z', 'K':
				session.Player.HandleUp()
			case keyA, keyQ, keyH, 'A', 'Q', 'H':
				session.Player.HandleLeft()
			case keyS, keyJ, keyO, 'S', 'J', 'O':
				session.Player.HandleDown()
			case keyD, keyL, keyE, 'D', 'L', 'E':
				session.Player.HandleRight()
			case keyCtrlC, keyEscape:
				g.removeSession(session, "~ End of Line ~")
				g.balanceBots(gm.Bots)
			}
			g.mu.Unlock()
		}

		// the connection is gone; it may already have been removed
		g.mu.Lock()
		g.removeSession(session, "")
		g.balanceBots(gm.Bots)
		g.mu.Unlock()
	}()
}

type Game struct {
	Name      string
	HighScore int

	// mu guards everything below and every player and session in the game
	mu       sync.Mutex
	sessions map[*Session]struct{}
	closed   bool

	// Top left is 0,0
	level [][]Tile
}

func NewGame(worldWidth, worldHeight int) *Game {
	g := &Game{
		Name:     petname.Generate(1, ""),
		sessions: make(map[*Session]struct{}),
	}
	g.initalizeLevel(worldWidth, worldHeight)

	return g
}

func (g *Game) initalizeLevel(width, height int) {
	g.level = make([][]Tile, width)
	for x := range g.level {
		g.level[x] = make([]Tile, height)
	}

	// Default world to grass
	for x := range g.level {
		for y := range g.level[x] {
			g.setTileType(Position{float64(x), float64(y)}, TileGrass)
		}
	}
}

func (g *Game) setTileType(pos Position, tileType TileType) error {
	outOfBoundsErr := "The given %s value (%v) is out of bounds"
	if pos.RoundX() >= len(g.level) || pos.RoundX() < 0 {
		return fmt.Errorf(outOfBoundsErr, "X", pos.X)
	} else if pos.RoundY() >= len(g.level[pos.RoundX()]) || pos.RoundY() < 0 {
		return fmt.Errorf(outOfBoundsErr, "Y", pos.Y)
	}

	g.level[pos.RoundX()][pos.RoundY()].Type = tileType

	return nil
}

func (g *Game) players() map[*Player]*Session {
	players := make(map[*Player]*Session)

	for session := range g.sessions {
		players[session.Player] = session
	}

	return players
}

// Characters for rendering
const (
	verticalWall   = '║'
	horizontalWall = '═'
	topLeft        = '╔'
	topRight       = '╗'
	bottomRight    = '╝'
	bottomLeft     = '╚'

	grass   = ' '
	blocker = '■'
)

// worldGrid draws the world as s's player sees it, one string per cell,
// indexed [x][y]. Each cell carries its own colour codes.
func (g *Game) worldGrid(s *Session) [][]string {
	worldWidth := len(g.level)
	worldHeight := len(g.level[0])

	// Create two dimensional slice of strings to represent the world. It's two
	// characters larger in each direction to accomodate for walls.
	strWorld := make([][]string, worldWidth+2)
	for x := range strWorld {
		strWorld[x] = make([]string, worldHeight+2)
	}

	// Load the walls into the rune slice
	borderColorizer := color.New(playerBorderColors[s.Player.Color]).SprintFunc()
	for x := 0; x < worldWidth+2; x++ {
		strWorld[x][0] = borderColorizer(string(horizontalWall))
		strWorld[x][worldHeight+1] = borderColorizer(string(horizontalWall))
	}
	for y := 0; y < worldHeight+2; y++ {
		strWorld[0][y] = borderColorizer(string(verticalWall))
		strWorld[worldWidth+1][y] = borderColorizer(string(verticalWall))
	}

	// Time for the edges!
	strWorld[0][0] = borderColorizer(string(topLeft))
	strWorld[worldWidth+1][0] = borderColorizer(string(topRight))
	strWorld[worldWidth+1][worldHeight+1] = borderColorizer(string(bottomRight))
	strWorld[0][worldHeight+1] = borderColorizer(string(bottomLeft))

	// Draw the player's score
	scoreStr := fmt.Sprintf(
		" Score: %d : Your High Score: %d : Game High Score: %d ",
		s.Player.Score(),
		s.HighScore,
		g.HighScore,
	)
	for i, r := range []rune(scoreStr) {
		strWorld[3+i][0] = borderColorizer(string(r))
	}

	// Draw the player's color
	colorStr := fmt.Sprintf(" %s ", playerColorNames[s.Player.Color])
	colorStrColorizer := color.New(s.Player.Color).SprintFunc()
	for i, r := range colorStr {
		charsRemaining := len(colorStr) - i
		strWorld[len(strWorld)-3-charsRemaining][0] = colorStrColorizer(string(r))
	}

	// Draw the game's name
	nameStr := fmt.Sprintf(" %s ", g.Name)
	for i, r := range nameStr {
		charsRemaining := len(nameStr) - i
		strWorld[len(strWorld)-3-charsRemaining][len(strWorld[0])-1] =
			borderColorizer(string(r))
	}
	// the scores below stop short of the name
	scoresEnd := len(strWorld) - 3 - len(nameStr) - 1

	// Draw everyone's scores
	if len(g.sessions) > 1 {
		// Sort the players by color name
		players := []*Player{}

		for player := range g.players() {
			if player == s.Player {
				continue
			}

			players = append(players, player)
		}

		sort.Sort(ByColor(players))
		startX := 3

		// Actually draw their scores
		for _, player := range players {
			colorizer := color.New(player.Color).SprintFunc()
			scoreStr := fmt.Sprintf(" %s: %d",
				playerColorNames[player.Color],
				player.Score(),
			)
			if startX+len(scoreStr) >= scoresEnd {
				break
			}
			for _, r := range scoreStr {
				strWorld[startX][len(strWorld[0])-1] = colorizer(string(r))
				startX++
			}
		}

		// Add final spacing next to wall
		strWorld[startX][len(strWorld[0])-1] = " "
	} else {
		warning :=
			" Warning: Other Players Must be in This Game for You to Score! "
		for i, r := range warning {
			strWorld[3+i][len(strWorld[0])-1] = borderColorizer(string(r))
		}
	}

	// Load the level into the string slice
	for x := 0; x < worldWidth; x++ {
		for y := 0; y < worldHeight; y++ {
			tile := g.level[x][y]

			switch tile.Type {
			case TileGrass:
				strWorld[x+1][y+1] = string(grass)
			case TileBlocker:
				strWorld[x+1][y+1] = string(blocker)
			}
		}
	}

	// Load the players into the rune slice
	for player := range g.players() {

		pos := player.Pos
		colorizer := color.New(player.Color).SprintFunc()
		strWorld[pos.RoundX()+1][pos.RoundY()+1] = colorizer(string(player.Marker))

		// Load the player's trail into the rune slice
		for _, segment := range player.Trail {
			x, y := segment.Pos.RoundX()+1, segment.Pos.RoundY()+1
			colorizer := color.New(segment.Color).SprintFunc()
			strWorld[x][y] = colorizer(string(segment.Marker))
		}
	}

	return strWorld
}

func (g *Game) WorldWidth() int {
	return len(g.level)
}

func (g *Game) WorldHeight() int {
	return len(g.level[0])
}

func (g *Game) availableColors() []color.Attribute {
	usedColors := map[color.Attribute]bool{}
	for _, color := range playerColors {
		usedColors[color] = false
	}

	for player := range g.players() {
		usedColors[player.Color] = true
	}

	availableColors := []color.Attribute{}
	for color, used := range usedColors {
		if !used {
			availableColors = append(availableColors, color)
		}
	}

	return availableColors
}

// humanCount counts the players at a terminal, an autopiloted one included
func (g *Game) humanCount() int {
	n := 0
	for s := range g.sessions {
		if s.c != nil {
			n++
		}
	}
	return n
}

func (g *Game) Run() {
	tick := time.NewTicker(time.Second / 60)
	defer tick.Stop()

	lastUpdate := time.Now()
	frame := 0
	for now := range tick.C {
		g.mu.Lock()
		if g.closed {
			g.mu.Unlock()
			return
		}

		g.Update(float64(now.Sub(lastUpdate)) / float64(time.Millisecond))
		lastUpdate = now

		frame++
		if frame%(60/framesPerSecond) == 0 {
			for s := range g.sessions {
				if s.c != nil {
					s.queue(g.worldGrid(s))
				}
			}
		}
		g.mu.Unlock()
	}
}

// Update is the main game logic loop. Delta is the time since the last update
// in milliseconds.
func (g *Game) Update(delta float64) {
	g.steerBots()

	// We'll use this to make a set of all of the coordinates that are occupied by
	// trails
	trailCoordMap := make(map[string]*PlayerTrailSegment)

	// Update player data
	for player, session := range g.players() {
		player.Update(g, delta)

		// Update session high score, if applicable
		if player.Score() > session.HighScore {
			session.HighScore = player.Score()
		}

		// Update global high score, if applicable
		if player.Score() > g.HighScore {
			g.HighScore = player.Score()
		}

		// Restart the player if they're out of bounds
		pos := player.Pos
		if pos.RoundX() < 0 || pos.RoundX() >= len(g.level) ||
			pos.RoundY() < 0 || pos.RoundY() >= len(g.level[0]) {
			session.StartOver(g.WorldWidth(), g.WorldHeight())
		}

		// Kick the player if they've timed out
		if session.timesOut() && time.Now().Sub(session.LastAction) > playerTimeout {
			g.removeSession(session, "You were terminated due to inactivity")
			return
		}

		// range gives copies, but we need a reference in the trailCoordMap so we
		// can modify the Color value if there is a collision, so iterate by index
		// instead.
		for i := range player.Trail {
			seg := &player.Trail[i]
			coordStr := fmt.Sprintf("%d,%d", seg.Pos.RoundX(), seg.Pos.RoundY())
			trailCoordMap[coordStr] = seg
		}
	}

	// Check if any players collide with a trail and restart them if so
	for player, session := range g.players() {
		playerPos := fmt.Sprintf("%d,%d", player.Pos.RoundX(), player.Pos.RoundY())
		if segment, collided := trailCoordMap[playerPos]; collided {
			segment.Color = player.Color
			session.StartOver(g.WorldWidth(), g.WorldHeight())
		}
	}
}

func (g *Game) addSession(s *Session) {
	g.sessions[s] = struct{}{}
	if s.c != nil {
		go s.writeFrames()
	}
}

// removeSession takes s out of the game and ends its connection with a last
// message. The game closes when no one at a terminal is left in it.
func (g *Game) removeSession(s *Session, message string) {
	if _, ok := g.sessions[s]; !ok {
		return
	}
	delete(g.sessions, s)
	s.end(message)
	if g.humanCount() == 0 {
		g.closed = true
	}
}

type Session struct {
	c    io.ReadWriteCloser
	size *termSize

	local     bool
	autopilot bool
	bot       *Bot

	LastAction time.Time
	HighScore  int
	Player     *Player

	frames   chan [][]string
	quit     chan string
	quitOnce sync.Once
}

func NewSession(conn Conn, worldWidth, worldHeight int,
	color color.Attribute) *Session {

	s := Session{
		c:          conn.c,
		size:       conn.size,
		local:      conn.local,
		autopilot:  conn.autopilot,
		LastAction: time.Now(),
		frames:     make(chan [][]string, 1),
		quit:       make(chan string, 1),
	}
	s.newGame(worldWidth, worldHeight, color)

	return &s
}

func (s *Session) newGame(worldWidth, worldHeight int, color color.Attribute) {
	s.Player = NewPlayer(s, worldWidth, worldHeight, color)
}

func (s *Session) didAction() {
	s.LastAction = time.Now()
}

func (s *Session) StartOver(worldWidth, worldHeight int) {
	s.newGame(worldWidth, worldHeight, s.Player.Color)
	if s.bot != nil {
		*s.bot = Bot{}
	}
}

// timesOut tells whether the inactivity timeout applies: to players who
// joined over the network, not to computer players or the host's own player
func (s *Session) timesOut() bool {
	return s.c != nil && !s.local && s.bot == nil
}

// queue hands the writer the newest frame, dropping one it has not sent yet.
// Only the game loop calls it.
func (s *Session) queue(grid [][]string) {
	select {
	case <-s.frames:
	default:
	}
	s.frames <- grid
}

func (s *Session) end(message string) {
	s.quitOnce.Do(func() { s.quit <- message })
}

// writeFrames sends each frame as the changes from the one before, centred
// in the player's terminal.
func (s *Session) writeFrames() {
	// Hide the cursor
	io.WriteString(s.c, "\033[?25l")

	var prev [][]string
	prevCols, prevRows := -1, -1
	tooSmall := false

	for {
		select {
		case message := <-s.quit:
			var b strings.Builder
			b.WriteString("\033[0m\033[2J\033[H")
			if message != "" {
				b.WriteString("\r\n" + message + "\r\n\r\n")
			}
			// Unhide the cursor
			b.WriteString("\033[?25h")
			io.WriteString(s.c, b.String())
			s.c.Close()
			return

		case grid := <-s.frames:
			w, h := len(grid), len(grid[0])
			cols, rows := s.size.get()
			var b strings.Builder

			if cols < w || rows < h {
				if !tooSmall || cols != prevCols || rows != prevRows {
					fmt.Fprintf(&b, "\033[0m\033[2J\033[H"+
						"SSHTron needs a terminal of %dx%d; this one is %dx%d.\r\n",
						w, h, cols, rows)
				}
				tooSmall = true
				prev = nil
			} else {
				if tooSmall || cols != prevCols || rows != prevRows {
					b.WriteString("\033[0m\033[2J")
					prev = nil
				}
				tooSmall = false
				ox, oy := (cols-w)/2, (rows-h)/2
				for y := 0; y < h; y++ {
					cursorAt := -1
					for x := 0; x < w; x++ {
						if prev != nil && prev[x][y] == grid[x][y] {
							continue
						}
						if cursorAt != x {
							fmt.Fprintf(&b, "\033[%d;%dH", oy+y+1, ox+x+1)
						}
						b.WriteString(grid[x][y])
						cursorAt = x + 1
					}
				}
				prev = grid
			}
			prevCols, prevRows = cols, rows

			if b.Len() > 0 {
				if _, err := io.WriteString(s.c, b.String()); err != nil {
					s.c.Close()
					return
				}
			}
		}
	}
}

// termSize is the size of a player's terminal, from the client's pty-req and
// window-change requests. Without them it stays at 80x24.
type termSize struct {
	mu         sync.Mutex
	cols, rows int
}

func newTermSize() *termSize {
	return &termSize{cols: 80, rows: 24}
}

func (t *termSize) set(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	t.mu.Lock()
	t.cols, t.rows = cols, rows
	t.mu.Unlock()
}

func (t *termSize) get() (int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cols, t.rows
}
