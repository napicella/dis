#!/usr/bin/env bash
# Search herdr shortcuts by description and print the keys. Runs in a herdr popup
# (see [[keys.command]] in config.toml) and reads the live config: active [keys]
# lines are custom bindings, commented ones are herdr's defaults, and a
# "# desc:" line above a binding is its description (else the action name).
set -euo pipefail

cfg=${1:-$HOME/.config/herdr/config.toml}

# Prints "description<TAB>keys" per binding, sorted by description.
list_keys() {
  awk '
    function value(line) { match(line, /"[^"]*"/); return substr(line, RSTART + 1, RLENGTH - 2) }

    function add(line, active) {
      split(line, kv, " *= *"); a = kv[1]
      if (pending != "") info[a] = pending
      if (active || !(a in custom)) bind[a] = value(line)
      if (active) custom[a] = 1
    }

    /^\[\[?keys(\.command)?\]\]?/ { section = $0; pending = ""; if (section ~ /command/) n++; next }
    /^\[/ { section = ""; next }
    # A "# desc:" line describes the binding right below it.
    section == "[keys]" && /^# desc: / { pending = substr($0, 9); next }
    # An active line overrides the commented default of the same action.
    section == "[keys]" && /^[a-z_]+ *= *"/ { add($0, 1); pending = ""; next }
    section == "[keys]" && /^# [a-z_]+ = "/ { add(substr($0, 3), 0); pending = ""; next }
    section == "[keys]" { pending = ""; next }
    section == "[[keys.command]]" && /^key *=/ { cmd_key[n] = value($0) }
    section == "[[keys.command]]" && /^description *=/ { cmd_desc[n] = value($0) }

    END {
      prefix = ("prefix" in bind) ? bind["prefix"] : "ctrl+b"
      for (a in bind) {
        if (bind[a] == "") continue
        if (a in info) desc = info[a]; else { desc = a; gsub(/_/, " ", desc) }
        keys = bind[a]; sub(/^prefix\+/, prefix " ", keys)
        printf "%s\t%s\n", desc, keys
      }
      for (i = 1; i <= n; i++) {
        keys = cmd_key[i]; sub(/^prefix\+/, prefix " ", keys)
        desc = (i in cmd_desc) ? cmd_desc[i] : "custom command"
        printf "%s\t%s\n", desc, keys
      }
    }
  ' "$cfg" | sort
}

rows=$(list_keys | awk -F'\t' '{ printf "%-52s %s\n", $1, $2 }')
[[ -n "$rows" ]] || { echo "No herdr shortcuts found in $cfg" >&2; exit 1; }

if command -v gum &> /dev/null; then
  pick=$(gum filter --placeholder "search herdr shortcuts" --height 20 <<< "$rows") || exit 0
else
  read -rp "search herdr shortcuts: " query
  pick=$(grep -i -- "$query" <<< "$rows" || true)
fi
[[ -n "$pick" ]] || exit 0

echo
echo "$pick"
echo
read -rsn1 -p "press any key to close" || true
