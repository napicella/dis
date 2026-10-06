#!/bin/bash
### -- Manifest
### provides: test/rc-tools
### distro: [test]
### -- End

# Verify that dis tools RC helpers write the correct sections to the generated RC files.

dis tools add-rc-init \
  --name 'test-init' \
  --content 'export TEST_INIT=1'

dis tools add-rc-env \
  --name 'test-env' \
  --content 'export TEST_ENV=from-bash-env
export TEST_ENV_BIN=/test/env-bin'

dis tools add-rc-path \
  --name 'test-path' \
  --path '/test/bin'

# bash_env is sourced before bash_paths, so a PATH entry can use its variables.
dis tools add-rc-path \
  --name 'test-env-path' \
  --path '$TEST_ENV_BIN'

dis tools add-rc-aliases \
  --name 'test-aliases' \
  --content 'alias ll="ls -la"'

touch /tmp/rc-tools-ran
