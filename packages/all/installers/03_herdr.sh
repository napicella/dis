### -- Manifest
### provides: common/herdr
### depends_on: []
### distro: [all]
### -- End

# https://herdr.dev - installed to ~/.local/bin by its own installer, and kept current
# with `herdr update`.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  mkdir -p ~/.config/herdr
  curl -fsSL https://herdr.dev/install.sh | sh
fi

# MIGRATION(2026-10-01): one-time cleanup; drop once every host has run dis config since then.
# herdr used to be in the global mise config as well. mise's copy came first on PATH,
# and `herdr update` replaced its binary behind mise's back. Drop it once. unuse only
# edits the config, so also uninstall mise's copy, which removes its shim too.
if command -v mise &> /dev/null && mise config get --global tools.herdr &> /dev/null; then
  mise unuse --global herdr
  mise uninstall --all herdr || true
fi

mkdir -p ~/.claude/hooks && cp "$DIS_CONFIG_FOLDER/herdr/herdr-claude-title.sh" ~/.claude/hooks/
bash "$DIS_CONFIG_FOLDER/herdr/install-claude-title-hook.sh"
mkdir -p ~/.claude/skills && cp -r "$DIS_CONFIG_FOLDER/herdr/spin-chat-herdr" ~/.claude/skills/
mkdir -p ~/.config/herdr && cp "$DIS_CONFIG_FOLDER/herdr/herdr-keys.sh" ~/.config/herdr/

# Keep the theme wal-picker set in the deployed copy ([theme]); a plain copy resets it.
# The [theme] block (with its [theme.*] subtables) is spliced in as text: a yq
# rewrite drops the comments after each table's last key, like the "# desc:" lines.
_cfg=~/.config/herdr/config.toml
_theme_block() { awk '/^\[/ { in_theme = ($0 ~ /^\[theme[].]/) } in_theme' "$1"; }
_theme=
if [[ -f "$_cfg" ]]; then
  _theme=$(_theme_block "$_cfg")
fi
mkdir -p ~/.config/herdr && cp "$DIS_CONFIG_FOLDER/herdr/config.toml" "$_cfg"
if [[ -n "$_theme" ]]; then
  theme="$_theme" awk '
    /^\[/ { in_theme = ($0 ~ /^\[theme[].]/); if (in_theme && !done) { print ENVIRON["theme"] "\n"; done = 1 } }
    !in_theme
  ' "$_cfg" > "$_cfg.tmp" && mv "$_cfg.tmp" "$_cfg"
fi

