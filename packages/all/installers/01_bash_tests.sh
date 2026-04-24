#!/bin/bash

### -- Manifest
### provides: common/bats-test
### depends_on: [common/os-libs]
### distro: [all]
### -- End

source $DIS_BINDING

# Install bash tests: https://github.com/bats-core/bats-core
# The executable is named bats, so we rename it to bash-test to avoid conflicts and for clarity

# Create a temporary directory
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

git clone https://github.com/bats-core/bats-core.git "$tmp_dir"
"$tmp_dir/install.sh" "$HOME/.local"

# Rename the bats executable to bash-test
mv "$HOME/.local/bin/bats" "$HOME/.local/bin/bash-test"