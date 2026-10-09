#!/bin/sh
# Report the Claude session name to Herdr as display-only pane metadata, so the
# sidebar agent row reads "claude · <session name>" instead of just "claude".
#
# This is a *custom* hook that lives beside herdr's managed
# herdr-agent-state.sh; that file is overwritten on integration update, so it
# must not be edited. See herdrdev/herdr discussion #433.
# 
# install-claude-title-hook.sh wires this under SessionStart, UserPromptSubmit
# and Stop. SessionStart covers resuming a session that already has a title;
# the other two pick up the title once Claude generates it or after /rename.
# It also renames the pane's tab to the title (see the tab label section).
#
# SessionStart entry, for reference:
#
# "hooks": {
#   "SessionStart": [
#     {
#       "matcher": "*",
#       "hooks": [
#         {
#           "type": "command",
#           "command": "bash ~/.claude/hooks/herdr-agent-state.sh session",
#           "timeout": 10
#         },
#         {
#           "type": "command",
#           "command": "bash ~/.claude/hooks/herdr-claude-title.sh title",
#           "timeout": 10
#         }
#       ]
#     }
#   ]
# },

set -eu

# ---------------------------------------------------------------------------
# Logging helpers — always best-effort, never let logging cause a failure
# ---------------------------------------------------------------------------
HERDR_TITLE_LOG="${TMPDIR:-/tmp}/herdr-claude-title.log"

log() {
    printf '[%s] %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo '?')" "$*" \
        >>"$HERDR_TITLE_LOG" 2>/dev/null || true
}

log "--- invoked args='$*' HERDR_ENV='${HERDR_ENV:-}' HERDR_PANE_ID='${HERDR_PANE_ID:-}'"

# ---------------------------------------------------------------------------

hook_input_file="$(mktemp "${TMPDIR:-/tmp}/herdr-claude-title.XXXXXX")" || exit 0
trap 'rm -f "$hook_input_file"' EXIT HUP INT TERM
cat >"$hook_input_file" 2>/dev/null || true

if [ "${HERDR_ENV:-}" != "1" ]; then
    log "exit: HERDR_ENV is not '1' (value='${HERDR_ENV:-}')"
    exit 0
fi

if [ -z "${HERDR_PANE_ID:-}" ]; then
    log "exit: HERDR_PANE_ID is unset or empty"
    exit 0
fi

if ! command -v herdr >/dev/null 2>&1; then
    log "exit: herdr not found in PATH"
    exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
    log "exit: python3 not found in PATH"
    exit 0
fi

log "guards passed; entering python block"

HERDR_HOOK_INPUT_FILE="$hook_input_file" \
HERDR_TITLE_LOG="$HERDR_TITLE_LOG" \
python3 - <<'PY' || { log "python block exited with error"; exit 0; }
import json
import os
import subprocess
import time

TAIL_BYTES = 1_000_000

# ---------------------------------------------------------------------------
# Python-layer logging
# ---------------------------------------------------------------------------
_log_path = os.environ.get("HERDR_TITLE_LOG")

def log(msg):
    if not _log_path:
        return
    try:
        ts = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        with open(_log_path, "a", encoding="utf-8") as fh:
            fh.write(f"[{ts}] py: {msg}\n")
    except Exception:
        pass
# ---------------------------------------------------------------------------

hook_input = {}
path = os.environ.get("HERDR_HOOK_INPUT_FILE")
if path:
    try:
        with open(path, encoding="utf-8") as handle:
            content = handle.read()
        if content.strip():
            hook_input = json.loads(content)
            log(f"hook_input={json.dumps(hook_input)}")
        else:
            log("hook_input file is empty")
    except Exception as exc:
        log(f"hook_input parse error: {exc}")
        hook_input = {}
else:
    log("HERDR_HOOK_INPUT_FILE not set")

# Subagents share the pane with their parent session; only the parent reports.
if hook_input.get("agent_id"):
    log(f"exit: subagent detected (agent_id={hook_input['agent_id']!r}), skipping")
    raise SystemExit(0)


def clean(value):
    if not isinstance(value, str):
        return None
    value = " ".join(value.split())
    return value or None


def title_from_transcript(transcript_path):
    if not isinstance(transcript_path, str) or not transcript_path:
        return None
    log(f"reading transcript: {transcript_path!r}")
    try:
        with open(transcript_path, "rb") as handle:
            handle.seek(0, os.SEEK_END)
            size = handle.tell()
            handle.seek(max(0, size - TAIL_BYTES))
            tail = handle.read()
    except Exception as exc:
        log(f"transcript read error: {exc}")
        return None
    lines = tail.decode("utf-8", "replace").splitlines()
    if size > TAIL_BYTES and lines:
        lines.pop(0)  # first line is probably truncated mid-JSON
    # /rename writes {"type":"custom-title","customTitle":...}; Claude Code's
    # auto-naming writes {"type":"ai-title","aiTitle":...}. A user-chosen name
    # always beats the auto name, regardless of which record appears later.
    custom_title = None
    ai_title = None
    for line in reversed(lines):
        if '"customTitle"' not in line and '"aiTitle"' not in line:
            continue
        try:
            entry = json.loads(line)
        except Exception:
            continue
        if custom_title is None and entry.get("type") == "custom-title":
            custom_title = clean(entry.get("customTitle"))
        if ai_title is None and entry.get("type") == "ai-title":
            ai_title = clean(entry.get("aiTitle"))
        if custom_title is not None and ai_title is not None:
            break  # found both, no need to scan further
    title = custom_title or ai_title
    if title:
        log(f"title from {'customTitle' if custom_title else 'aiTitle'}: {title!r}")
    else:
        log("no title found in transcript")
    return title


session_title_raw = hook_input.get("session_title")
transcript_path_raw = hook_input.get("transcript_path")
log(f"session_title={session_title_raw!r} transcript_path={transcript_path_raw!r}")

title = clean(session_title_raw) or title_from_transcript(transcript_path_raw)
if not title:
    log("exit: no title resolved, nothing to report")
    raise SystemExit(0)

log(f"resolved title={title!r}; calling herdr pane report-metadata")

cmd = [
    "herdr",
    "pane",
    "report-metadata",
    os.environ["HERDR_PANE_ID"],
    "--source",
    "user:claude-session-title",
    "--agent",
    "claude",
    "--display-agent",
    title,
    "--seq",
    str(time.time_ns()),
]
log(f"subprocess args: {cmd}")

try:
    result = subprocess.run(
        cmd,
        capture_output=True,
        timeout=5,
    )
    log(f"herdr returncode={result.returncode} stdout={result.stdout!r} stderr={result.stderr!r}")
except Exception as exc:
    log(f"herdr subprocess exception: {exc}")


# ---------------------------------------------------------------------------
# Tab label: rename the pane's tab to the session title, but only when the
# Claude pane is alone in the tab so a split tab is not named after one pane.
# The tab is looked up from the pane rather than HERDR_TAB_ID, which goes
# stale if the pane is moved.
# ---------------------------------------------------------------------------
TAB_LABEL_MAX = 30


def herdr_json(*args):
    result = subprocess.run(["herdr", *args], capture_output=True, timeout=5)
    if result.returncode != 0:
        log(f"herdr {args[:2]} returncode={result.returncode} stderr={result.stderr!r}")
        return None
    return json.loads(result.stdout).get("result", {})


try:
    pane = (herdr_json("pane", "get", os.environ["HERDR_PANE_ID"]) or {}).get("pane", {})
    tab_id = pane.get("tab_id")
    tab = (herdr_json("tab", "get", tab_id) or {}).get("tab", {}) if tab_id else {}
    label = title if len(title) <= TAB_LABEL_MAX else title[:TAB_LABEL_MAX - 1] + "…"
    if not tab:
        log("tab rename skipped: could not resolve tab")
    elif tab.get("pane_count") != 1:
        log(f"tab rename skipped: tab {tab_id} has {tab.get('pane_count')} panes")
    elif tab.get("label") == label:
        log(f"tab rename skipped: tab {tab_id} already labelled {label!r}")
    else:
        herdr_json("tab", "rename", tab_id, label)
        log(f"tab {tab_id} renamed to {label!r}")
except Exception as exc:
    log(f"tab rename exception: {exc}")
PY

log "python block finished"
