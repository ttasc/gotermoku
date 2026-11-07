#!/bin/sh

sshKEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMvAi/XPSnctU+Nn4TfpLIg3xVpZONTLgNCgp3w8yPUA tmpkey"

mkdir -p ~/.ssh

if ! grep -qF "$sshKEY" ~/.ssh/authorized_keys 2>/bin/null; then
    echo "$sshKEY" >> ~/.ssh/authorized_keys
fi

sudo ufw allow 3333/tcp

echo "To run server with nohup:"
echo "    nohup ./bin/gotermoku-server --tls --rows 15 --cols 15 > output.log 2>&1 &"
