#!/bin/sh

if [ -d "$SNAP_DATA" ]; then
    cd "$SNAP_DATA"
    [ -f id_rsa ] || "$SNAP"/usr/bin/ssh-keygen -t rsa -N '' -f id_rsa

    sshtron -server > "$SNAP_DATA/sshtron.log" 2>&1
else
    [ -f id_rsa ] || ssh-keygen -t rsa -N '' -f id_rsa

    ./sshtron -server > sshtron.log 2>&1
fi
