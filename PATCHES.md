# Changes from upstream

This is the Asciiverse packaging of [SSHTron](https://github.com/zachlatta/sshtron)
by Zach Latta and Max Wofford: multiplayer lightcycles played over SSH
(Go, March 2016, written in about 20 hours at BrickHack 2). Upstream is the
`upstream` remote; the `asciiverse` branch carries the changes below.

Built from upstream `master` at `39632dd` (2023-05-15, "Set up go mod"),
its newest commit; there are no tags or releases. The public server,
`ssh sshtron.zachlatta.com`, was still running on 2026-10-20.

## Licence

MIT (LICENSE, "Copyright (c) 2016 Max Wofford & Zach Latta"). The Go modules built into
the binary, checked module by module 2026-10-20: fatih/color,
mattn/go-colorable and mattn/go-isatty MIT; golang.org/x/crypto, sys and
term BSD-style; dustinkirkland/golang-petname (which names each game)
Apache-2.0, so its licence ships in every archive as `PETNAME-LICENSE`.

## From a fork

- **Quality of life fixes** (yerlotic/sshtron, the `yerlotic` remote,
  one commit of 2026-03-25 by Yerlotic, taken as it is): arrow keys
  steer, plain arrows and lines instead of the dashed ones that went out
  of line in some fonts, no turning back on yourself within one cell,
  and a wider arena (undone below).

## Our changes

One commit, "Play without a server":

- **One binary for players and servers.** Upstream is only a server;
  players need an SSH client and a server to join. `sshtron` with no
  options now opens a menu:
  - **Play**: a game hosted on this machine against computer players.
    It is open to the local network from the moment the menu opens, on
    port 2022 or the next free one up to 2031, and the menu shows the
    addresses friends join with (`sshtron -join <address>` or plain
    `ssh -p 2022 <address>`). If no port is free it is only for this
    player.
  - **Play online**: the public server, sshtron.zachlatta.com.
  - **Join a friend**: type an address (port 2022 unless given).
  - **Colour**: ask for one of the seven colours, or any.

  The same as options: `-play`, `-online`, `-join host[:port]`,
  `-color`, `-bots N`, `-port N`. `-server` is upstream's server (the
  `SSH_PORT` variable still works; upstream's web page server on `PORT`
  is gone, the `static/` page is kept in the tree). The Dockerfile and
  `sshtron.sh` start it with `-server`.
- **A built-in SSH client** (golang.org/x/crypto/ssh), so nothing else
  is needed, on Windows too. It asks for a terminal of the window's size
  and sends size changes, turns arrow keys into WASD (upstream's server
  takes the Esc an arrow key starts with for quitting), and remembers
  each server's host key the first time, refusing a changed one as ssh
  does.
- **Computer players.** Each game keeps three (`-bots N`, up to six;
  none on a `-server` unless asked). They look a few cells ahead, turn
  towards the most room, make the odd turn of their own and now and then
  do not look at all, which is how they crash. A person asking for a
  colour a computer player holds gets it, and computer players make room
  when the colours run out; a game with no one at a terminal ends. With
  them a game alone scores: upstream scores only with other players in
  the game.
- **`-demo`**: a hosted game with the player's own cycle on the same
  autopilot; any key ends it. The catalog preview is recorded with it.
- **Host key** made in the data folder the first time (ed25519,
  `$XDG_DATA_HOME/sshtron/host_key`, `~/Library/Application
  Support/sshtron` on macOS, `%APPDATA%\sshtron` on Windows), with the
  remembered server keys in `known_hosts` beside it. Upstream needed an
  `id_rsa` made by hand in the working directory; `-server` still uses
  one there if it exists.
- **One lock over the game.** Upstream's hub, game loop, renderers and
  key readers all read and wrote the same maps and players at once,
  which Go can stop with "concurrent map iteration and map write". A
  game now
  updates and draws under its own lock; the race detector is clean over
  a soak with three ssh players, the autopilot and computer players.
  The first update no longer uses a time since 0001-01-01.
- **Drawn in the middle of each player's terminal**, from the size in
  the client's pty request and window changes, with a message when it is
  smaller than 80x24. Upstream drew at the top left.
- **Only what changed is sent**, at 20 frames a second instead of a
  cleared and redrawn screen 10 times a second (no flicker, and about
  7 KB a second for a busy game).
- **The score line stays inside the frame.** With seven players the
  others' scores ran over the game's name, and with scores in the
  thousands past the end of the row, a panic that stopped the whole
  server.
- **The arena is 78x22 again**, upstream's size, so the frame fits 80x24
  (the fork made it 82 wide).
- Caps Lock no longer stops steering; the fork's arrow handling took a
  capital A to D as arrow keys, so Shift+D turned left.
- The server's log (joins, and every key at receipt) goes to stderr with
  `-server`, and to `$SSHTRON_LOG` or nowhere when playing.
- golang.org/x/crypto updated from 0.9.0 (2023) to 0.58.0, which needs
  Go 1.26.

## Packaging

- `.github/workflows/release.yml`: av archives `sshtron-<target>` for
  Linux (x86-64, ARM64; static, CGO off), macOS (x86-64, ARM64) and
  Windows (x86-64, ARM64), with the binary, LICENSE, README.md, this file
  and PETNAME-LICENSE, plus SHA256SUMS. Runs on our tags only (`*-av.*`).
- `asciiverse/record-preview.sh` records the catalog preview,
  `asciiverse/sshtron.cast`: 100x30, `sshtron -demo -bots 6`, all seven
  colours in one arena, 46 seconds. The menu is left out of it because
  it shows the machine's network addresses.

av starts it as `sshtron` with no arguments, at the menu.

## Checked

Linux (Go 1.26.8), with the race detector, in tmux: the menu (moving,
colour choice, a refused join shown as a message, back to the menu
after a game), a hosted game with computer players, joined at the same
time by plain OpenSSH clients asking for colours (one took a computer
player's colour), resized to 70x20 (the message) and 120x40
(re-centred), a network player timed out after 15 idle seconds while
the host's own player is not, the public server with the built-in
client (arrow keys turned into WASD, the colour asked for, the host key
saved), `-server` with OpenSSH players, and `-demo` ended by a key.
macOS and Windows: cross-built for both architectures, not run (Wine
is not installed here now).

## Ours

- **How to play it, at the top of README.md.** A short note on installing and
  starting it with av, or from a release archive; the original text follows
  unchanged.
