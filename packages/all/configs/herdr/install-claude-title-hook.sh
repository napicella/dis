#!/usr/bin/env bash
# install-claude-title-hook.sh
#
# Idempotently injects the herdr-claude-title.sh hook into
# ~/.claude/settings.json under SessionStart, Stop, and UserPromptSubmit.
#
# Safe to run multiple times — skips any hook entry that is already present.

set -euo pipefail

SETTINGS="${HOME}/.claude/settings.json"
HOOK_COMMAND="bash '${HOME}/.claude/hooks/herdr-claude-title.sh' title"

if [[ ! -f "$SETTINGS" ]]; then
  echo "install-claude-title-hook: $SETTINGS not found, skipping" >&2
  exit 0
fi

python3 - "$SETTINGS" "$HOOK_COMMAND" <<'PY'
import json
import os
import sys

settings_path = sys.argv[1]
hook_command   = sys.argv[2]

new_hook = {
    "type":    "command",
    "command": hook_command,
    "timeout": 10,
}

with open(settings_path, encoding="utf-8") as fh:
    settings = json.load(fh)

hooks_root = settings.setdefault("hooks", {})

# ------------------------------------------------------------------
# Helper: inject new_hook into every matcher-block in an event list.
# Returns True if any change was made.
# ------------------------------------------------------------------
def inject_into_event(event_list):
    changed = False
    for block in event_list:
        inner = block.setdefault("hooks", [])
        already = any(
            isinstance(h, dict) and h.get("command") == hook_command
            for h in inner
        )
        if not already:
            inner.append(new_hook)
            changed = True
    return changed

any_change = False

# SessionStart — add to the first matcher block (alongside herdr-agent-state.sh)
ss_list = hooks_root.setdefault("SessionStart", [])
if not ss_list:
    # No existing entry — create a fresh one
    ss_list.append({"matcher": "*", "hooks": []})
any_change |= inject_into_event([ss_list[0]])

# Stop
stop_list = hooks_root.setdefault("Stop", [])
if not stop_list:
    stop_list.append({"matcher": "*", "hooks": []})
any_change |= inject_into_event(stop_list)

# UserPromptSubmit
ups_list = hooks_root.setdefault("UserPromptSubmit", [])
if not ups_list:
    ups_list.append({"matcher": "*", "hooks": []})
any_change |= inject_into_event(ups_list)

if not any_change:
    print("install-claude-title-hook: hook already present in all events, nothing to do")
    sys.exit(0)

# Write back atomically
tmp_path = settings_path + ".tmp"
with open(tmp_path, "w", encoding="utf-8") as fh:
    json.dump(settings, fh, indent=2, ensure_ascii=False)
    fh.write("\n")
os.replace(tmp_path, settings_path)
print(f"install-claude-title-hook: updated {settings_path}")
PY
