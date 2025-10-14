#!/usr/bin/env bash

set -e

envsubst < fileserver.service.template | sudo tee /etc/systemd/system/fileserver.service > /dev/null

sudo systemctl daemon-reload
sudo systemctl enable fileserver.service
sudo systemctl start fileserver.service
