#!/bin/bash
# wrapper.sh
#
# Written to a temp file by the dis runner and used to wrap every installer
# script. It bootstraps the generated RC environment and then exec's the
# installer (or any other script) passed as its first argument.
#
# What this does:
#   1. Creates ~/rc/configs-generated/ and touches the four RC files so that
#      sourcing them is always safe even on a fresh machine.
#   2. Sources bash_env, bash_paths and bash_aliases so that exports (e.g.
#      GOBIN, EDITOR) and PATH additions written by earlier installers in the
#      same run are available to the current one. bash_env comes first so PATH
#      entries can use the variables it exports. bash_init is intentionally NOT
#      sourced here because it contains code designed for interactive sessions
#      only.
#   3. exec's the installer script passed as "$@".

set -e

RC_CFG_GEN_FOLDER="$HOME/rc/configs-generated"

mkdir -p "$RC_CFG_GEN_FOLDER"
touch "$RC_CFG_GEN_FOLDER/bash_env"
touch "$RC_CFG_GEN_FOLDER/bash_paths"
touch "$RC_CFG_GEN_FOLDER/bash_aliases"
touch "$RC_CFG_GEN_FOLDER/bash_init"

# shellcheck source=/dev/null
source "$RC_CFG_GEN_FOLDER/bash_env"
# shellcheck source=/dev/null
source "$RC_CFG_GEN_FOLDER/bash_paths"
# shellcheck source=/dev/null
source "$RC_CFG_GEN_FOLDER/bash_aliases"

exec /bin/bash -e "$@"
