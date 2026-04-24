#!/bin/bash
### -- Manifest
### provides: test/jq
### distro: [test]
### -- End
source "$DIS_BINDING"

sudo apt-get update -qq
sudo apt-get install -y -qq jq
