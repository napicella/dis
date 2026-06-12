#!/bin/bash
# wrapper.sh
#
# Written to a temp file by the dis runner and used to wrap every installer
# script. It bootstraps the generated RC environment and then exec's the
# installer (or any other script) passed as its first argument.
#
# What this does:
#   1. Creates ~/rc/configs-generated/ and touches the three RC files so that
#      sourcing them is always safe even on a fresh machine.
#   2. Sources bash_paths and bash_aliases so that PATH additions written by
#      earlier installers in the same run are available to the current one.
#      bash_init is intentionally NOT sourced here because it contains code
#      designed for interactive sessions only.
#   3. exec's the installer script passed as "$@".

set -e

RC_CFG_GEN_FOLDER="$HOME/rc/configs-generated"

mkdir -p "$RC_CFG_GEN_FOLDER"
touch "$RC_CFG_GEN_FOLDER/bash_paths"
touch "$RC_CFG_GEN_FOLDER/bash_init"
touch "$RC_CFG_GEN_FOLDER/bash_aliases"

# shellcheck source=/dev/null
source "$RC_CFG_GEN_FOLDER/bash_paths"
# shellcheck source=/dev/null
source "$RC_CFG_GEN_FOLDER/bash_aliases"


# Exports a KEY=VALUE pair to DIS_EXPORTS_FILE so that downstream installers
# that declare "provides:KEY" in requires_env can import it.
#
# Usage: dis_export KEY value
# Example: dis_export GID_DOCKER "$(getent group docker | cut -d: -f3)"
function dis_export() {
  local key="$1"
  local value="$2"
  if [ -z "${DIS_EXPORTS_FILE:-}" ]; then
    echo "dis_export: DIS_EXPORTS_FILE is not set" >&2
    return 1
  fi
  echo "${key}=${value}" >> "$DIS_EXPORTS_FILE"
}

export -f dis_export
exec /bin/bash -e "$@"
