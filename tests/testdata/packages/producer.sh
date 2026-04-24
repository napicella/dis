#!/bin/bash
### -- Manifest
### provides: test/producer
### distro: [test]
### exports_env: [TOKEN]
### -- End
source "$DIS_BINDING"

touch /tmp/producer-ran
echo "TOKEN=abc123" >> "$DIS_EXPORTS_FILE"
