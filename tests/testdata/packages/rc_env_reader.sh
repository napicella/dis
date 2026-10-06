#!/bin/bash
### -- Manifest
### provides: test/rc-env-reader
### distro: [test]
### depends_on: [test/rc-tools]
### -- End

# Runs after test/rc-tools: the wrapper sourced bash_env and bash_paths, so the
# export and the PATH entry built from it are visible here.
echo "$TEST_ENV" > /tmp/rc-env-seen
case ":$PATH:" in *":/test/env-bin:"*) echo yes ;; *) echo no ;; esac > /tmp/rc-env-path-seen
