#!/bin/bash
### -- Manifest
### provides: test/producer
### distro: [test]
### exports_env: [TOKEN]
### -- End
touch /tmp/producer-ran
echo "TOKEN=abc123" >> "$DIS_EXPORTS_FILE"
