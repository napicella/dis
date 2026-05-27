#!/bin/bash
### -- Manifest
### provides: test/pkg-a
### distro: [test]
### requires_env: [GLOBAL_PARAM, SHARED_PARAM]
### -- End
# Write injected env vars to a file so the test can inspect them.
echo "GLOBAL_PARAM=${GLOBAL_PARAM}" >> /tmp/pkg-a-env
echo "SHARED_PARAM=${SHARED_PARAM}" >> /tmp/pkg-a-env
touch /tmp/pkg-a-ran
