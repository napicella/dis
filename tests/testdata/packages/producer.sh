#!/bin/bash
### -- Manifest
### provides: test/producer
### distro: [test]
### exports_env: [TOKEN]
### -- End
touch /tmp/producer-ran
dis tools export-env --key TOKEN --value abc123