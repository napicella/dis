#!/usr/bin/env bash
### -- Manifest
### provides: common/yq
### depends_on: [common/os-libs]
### distro: [all]
### -- End

# Install https://github.com/mikefarah/yq
sudo wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/local/bin/yq && \
    sudo chmod +x /usr/local/bin/yq