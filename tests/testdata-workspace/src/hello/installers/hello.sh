#!/bin/bash
### -- Manifest
### provides: ws/hello
### distro: [test]
### -- End
source "$DIS_BINDING"

touch /tmp/ws-hello-ran
cat "$DIS_CONFIG_FOLDER/greeting.txt" > /tmp/ws-greeting
echo "$DIS_PKG_ROOT" > /tmp/ws-pkg-root
