#!/bin/bash
### -- Manifest
### provides: test/rc-tools
### distro: [test]
### -- End

# Verify that dis tools RC helpers write the correct sections to the generated RC files.

dis tools add-rc-init \
  --name 'test-init' \
  --content 'export TEST_INIT=1'

dis tools add-rc-path \
  --name 'test-path' \
  --content 'export PATH="/test/bin:$PATH"'

dis tools add-rc-aliases \
  --name 'test-aliases' \
  --content 'alias ll="ls -la"'

touch /tmp/rc-tools-ran
