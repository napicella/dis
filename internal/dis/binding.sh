# binding.sh — TOMBSTONE
#
# This file is kept to preserve git history but is NO LONGER USED.
#
# As of the wrapper.sh refactor, dis no longer sets DIS_BINDING or
# requires installers to source this file. The functionality has been
# split into two separate concerns:
#
#   1. RC bootstrapping (sourcing bash_paths / bash_aliases before each installer)
#      → now handled by internal/dis/wrapper.sh, which wraps every installer run.
#
#   2. RC helper functions (bashrc_init_add, bashrc_path_add, bashrc_aliases_add)
#      → now exposed as `dis tools add-rc-init`, `dis tools add-rc-path`,
#        `dis tools add-rc-aliases`, and `dis tools add-home-rc`.
#        Implemented in internal/tools/rc.go.
#
# Installers should call `dis tools <subcommand>` instead of sourcing this file.
