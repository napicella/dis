---
name: spin-chat-herdr
description: Spin off a separate Claude Code chat in a new Herdr tab with a self-contained prompt. Use when the user says "spin a new chat", "spin this off", "tackle that separately", "open a new tab for this", or invokes /spin-chat-herdr.
---

# Spin off a new chat in a Herdr tab

The new chat has NONE of this conversation's context. Your job is to hand off a task cleanly.

## 1. Check you are inside Herdr

```bash
test "${HERDR_ENV:-}" = 1
```

If this fails, tell the user you're not running inside Herdr and stop.

## 2. Write the handoff prompt

Draft a self-contained prompt for the new chat:
- The goal, in one or two sentences.
- Relevant context from this conversation: file paths, findings, decisions already made, constraints, what was ruled out.
- What "done" looks like.

Keep it tight. If the user supplied the prompt text themselves, use it as-is plus any context it clearly needs.

Also pick a short tab label and agent name (`[a-z][a-z0-9_-]{0,31}`, unique), e.g. `fix-proxy`.

## 3. Create the tab and start Claude

Use the current working directory unless the user asked for another one. Do not steal focus unless asked.

```bash
NAME=fix-proxy   # agent name / tab label
PANE=$(herdr tab create --workspace "$HERDR_WORKSPACE_ID" --cwd "$PWD" --label "$NAME" --no-focus \
  | jq -r '.result.root_pane.pane_id')
herdr agent start "$NAME" --kind claude --pane "$PANE"
```

## 4. Send the prompt

Pass the prompt through a quoted heredoc so quotes/backticks survive. Do NOT use `--wait`: the new chat runs independently.

```bash
herdr agent prompt "$NAME" "$(cat <<'EOF'
<handoff prompt here>
EOF
)"
```

If `agent start` returned `agent_not_ready` (e.g. a trust/approval dialog), run `herdr agent read "$NAME"` and tell the user the new tab needs their attention instead of answering the dialog yourself.

## 5. Report back

One line: tab label and a one-sentence summary of what was handed off. Then continue with the current conversation.
