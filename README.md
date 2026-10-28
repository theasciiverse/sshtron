> **Play it with Asciiverse.** With [av](https://asciiverse.sh), the Asciiverse
> client:
>
> ```
> av install sshtron
> av play sshtron
> ```
>
> The original README follows.

# ![SSHTron](https://cdn.rawgit.com/zachlatta/sshtron/master/logo.svg)

SSHTron is a multiplayer lightcycle game that runs through SSH. Just run the command below and you'll be playing in seconds:

    $ ssh sshtron.zachlatta.com

_Controls: WASD, arrow keys or vim keybindings to move. Escape or Ctrl+C to exit._

![Demo](static/img/gameplay.gif)

**Code quality disclaimer:** _SSHTron was built in ~20 hours at [BrickHack 2](https://brickhack.io/). Here be dragons._

## Want to choose color yourself?

There are total 7 colors to choose from: Red, Green, Yellow, Blue, Magenta, Cyan and White

    $ ssh red@sshtron.zachlatta.com

If the color you picked is already taken in all open games, you'll randomly be assigned a color.

## Playing without a server

`sshtron` on its own opens a menu: play a game hosted on your own machine
against computer players (friends on your network join it with
`sshtron -join <your address>` or `ssh -p 2022 <your address>`), play on
the public server, or join a friend's game. It carries its own SSH client,
so nothing else is needed, on Windows too. `sshtron -help` lists the
options.

## Running Your Own Copy

```sh
# Compile the project
$ go build

# Run the server. It makes a host key in its data folder the first time
# (an id_rsa in the current directory is used instead if there is one).
# -port or SSH_PORT sets the SSH port (2022), -bots N keeps N computer
# players in each game.
$ ./sshtron -server
```

## Running under a Docker container

Clone the project and `cd` into its directory.

```sh
# Build the SSHTron Docker image
$ docker build -t sshtron .

# Spin up the container with always-restart policy
$ docker run -t -d -p 2022:2022 --restart always --name sshtron sshtron
```

For Raspberry Pi, use the following to build the Docker image:

```sh
$ docker build -t sshtron --build-arg BASE_IMAGE=resin/raspberry-pi-golang:latest .
```

## CVE-2016-0777

[CVE-2016-0777](https://www.qualys.com/2016/01/14/cve-2016-0777-cve-2016-0778/openssh-cve-2016-0777-cve-2016-0778.txt)
revealed two SSH client vulnerabilities that can be exploited by a malicious SSH server. While SSHTron does not exploit
these vulnerabilities, you should still patch your client before you play. SSHTron is open source, but the server
could always be running a modified version of SSHTron that does exploit the vulnerabilities described
in [CVE-2016-0777](https://www.qualys.com/2016/01/14/cve-2016-0777-cve-2016-0778/openssh-cve-2016-0777-cve-2016-0778.txt).

If you haven't yet patched your SSH client, you can follow
[these instructions](https://www.jacobtomlinson.co.uk/quick%20tip/2016/01/15/fixing-ssh-vulnerability-CVE-2016-0777/) to do so now.

## License

SSHTron is licensed under the MIT License. See the full license text in [`LICENSE`](LICENSE).
