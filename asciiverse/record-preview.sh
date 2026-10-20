#!/bin/sh
# Records the catalog preview, asciiverse/sshtron.cast, to hub/RECORDING.md:
# 100x30, a hosted game watched in -demo mode (the player's own cycle on
# autopilot) against six computer players, all seven colours, then a key
# to end it.
# Runs asciinema in a detached tmux pane, so nothing from this terminal
# gets in; a throwaway HOME keeps the host key and known hosts untouched.
# The hosted game listens on 2022 (or the next free port) meanwhile.
#
#   go build && asciiverse/record-preview.sh
set -eu
cd "$(dirname "$0")/.."
out=asciiverse/sshtron.cast
tmp=$(mktemp -d)
trap 'tmux kill-session -t sshtron-rec 2>/dev/null || true; rm -rf "$tmp"' EXIT
rm -f "$out"
tmux new-session -d -s sshtron-rec -x 100 -y 30 \
  "env -i PATH=$PWD:/usr/bin:/bin HOME=$tmp USER=player LOGNAME=player \
     TERM=xterm-256color COLORTERM=truecolor LANG=C.UTF-8 \
   asciinema rec --window-size 100x30 --idle-time-limit 2 \
     --capture-env TERM,COLORTERM,LANG --title SSHTron \
     --command 'sshtron -demo -bots 6 -color cyan' $out"
sleep 46
tmux send-keys -t sshtron-rec q
sleep 2
ls -la "$out"
