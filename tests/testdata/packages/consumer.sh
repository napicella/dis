#!/bin/bash
### -- Manifest
### provides: test/consumer
### distro: [test]
### depends_on: [test/producer]
### requires_env: [test/producer:TOKEN,GREETING]
### -- End
touch /tmp/consumer-ran
echo "$TOKEN" > /tmp/token
echo "$GREETING" > /tmp/greeting
