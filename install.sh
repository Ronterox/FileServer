#!/usr/bin/env bash

set -e

# Check if Go is installed
if ! command -v go &> /dev/null
then
    echo "Go is not installed. Installing..."
    sudo snap install go
fi

mkdir -p "$HOME/FileServer"

go build -o "$HOME/FileServer/fileserver"

envsubst < fileserver.service.template | sudo tee /etc/systemd/system/fileserver.service > /dev/null

sudo systemctl daemon-reload
sudo systemctl enable fileserver.service
sudo systemctl start fileserver.service
