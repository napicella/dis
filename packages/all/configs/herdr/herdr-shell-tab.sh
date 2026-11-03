#!/usr/bin/env bash
# Focus the tab labelled "shell" in the focused workspace, creating it when the
# workspace has none. Runs in the background from a herdr key (see
# [[keys.command]] in config.toml); pass another label to jump to that tab instead,
# and --no-create to do nothing when the workspace has no such tab.
set -euo pipefail

label=${1:-shell}
create=1
[[ "${2:-}" == --no-create ]] && create=0
herdr=${HERDR_BIN_PATH:-$HOME/.local/bin/herdr}

workspace=$("$herdr" workspace list | jq -r '.result.workspaces[] | select(.focused) | .workspace_id')
[[ -n "$workspace" ]] || exit 0

tab=$("$herdr" tab list --workspace "$workspace" |
  jq -r --arg want "$label" 'first(.result.tabs[] | select(.label == $want) | .tab_id) // empty')

if [[ -n "$tab" ]]; then
  "$herdr" tab focus "$tab" > /dev/null
elif (( create )); then
  "$herdr" tab create --workspace "$workspace" --label "$label" --focus > /dev/null
fi
