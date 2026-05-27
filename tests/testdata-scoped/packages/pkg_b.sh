#!/bin/bash
### -- Manifest
### provides: test/pkg-b
### distro: [test]
### requires_env: [GLOBAL_PARAM, SHARED_PARAM, EXCLUSIVE_PARAM]
### -- End
# Write injected env vars to a file so the test can inspect them.
echo "GLOBAL_PARAM=${GLOBAL_PARAM}" >> /tmp/pkg-b-env
echo "SHARED_PARAM=${SHARED_PARAM}" >> /tmp/pkg-b-env
echo "EXCLUSIVE_PARAM=${EXCLUSIVE_PARAM}" >> /tmp/pkg-b-env
touch /tmp/pkg-b-ran
