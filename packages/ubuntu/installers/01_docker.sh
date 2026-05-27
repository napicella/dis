#!/bin/bash

### -- Manifest
### provides: common/docker
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End


# Add the official Docker repo
sudo install -m 0755 -d /etc/apt/keyrings
sudo wget -qO /etc/apt/keyrings/docker.asc https://download.docker.com/linux/ubuntu/gpg
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update

# Install Docker engine and standard plugins
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin docker-ce-rootless-extras

# Create the docker group if it does not exist (normally creates by the apt packages installers)
sudo groupadd docker | true
# Give this user privileged Docker access
sudo usermod -aG docker ${USER}
# TODO: `newgrp` starts a new interactive shell and waits for input — it never exits when run non-interactively!!!!
# Activate the changes to the group
# newgrp docker

# Limit log size to avoid running out of disk
echo '{"log-driver":"json-file","log-opts":{"max-size":"10m","max-file":"5"}}' | sudo tee /etc/docker/daemon.json

sudo systemctl stop docker
sudo systemctl daemon-reload
sudo systemctl start docker